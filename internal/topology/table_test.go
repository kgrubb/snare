package topology

import (
	"testing"
	"time"

	"github.com/bingoohuang/gosnmpd"
	"github.com/kgrubb/snare/internal/config"
	"github.com/slayercat/gosnmp"
)

func TestNewDefaults(t *testing.T) {
	t.Parallel()

	tab := New([]config.Neighbor{
		{Port: 2, SysName: "peer-a"},
		{Port: 1, ChassisID: "aa:bb:cc:dd:ee:ff", PortID: "eth0"},
	})
	n1, ok := tab.Get(1)
	if !ok || n1.ChassisIDSubtype != 7 || n1.PortIDSubtype != 5 {
		t.Fatalf("port1 = %+v ok=%v", n1, ok)
	}
	n2, ok := tab.Get(2)
	if !ok || n2.ChassisIDSubtype != 7 || len(tab.Ports()) != 2 {
		t.Fatalf("port2 = %+v ports=%v", n2, tab.Ports())
	}
}

func TestSplitAddr(t *testing.T) {
	t.Parallel()
	host, port := splitAddr("192.0.2.1:1161")
	if host != "192.0.2.1" || port != 1161 {
		t.Fatalf("got %s:%d", host, port)
	}
	host, port = splitAddr("192.0.2.1")
	if host != "192.0.2.1" || port != 161 {
		t.Fatalf("bare host got %s:%d", host, port)
	}
}

func startPeer(t *testing.T, community string, oids map[string]string) *gosnmpd.SNMPServer {
	t.Helper()
	items := make([]*gosnmpd.PDUValueControlItem, 0, len(oids))
	for oid, val := range oids {
		oid, val := oid, val
		items = append(items, &gosnmpd.PDUValueControlItem{
			OID:  oid,
			Type: gosnmp.OctetString,
			OnGet: func() (any, error) {
				return gosnmpd.Asn1OctetStringWrap(val), nil
			},
		})
	}
	srv := gosnmpd.NewSNMPServer(gosnmpd.MasterAgent{
		SecurityConfig: gosnmpd.SecurityConfig{
			AuthoritativeEngineBoots: 1,
			AuthoritativeEngineID:    gosnmpd.SNMPEngineID{EngineIDData: "snare-test"},
		},
		SubAgents: []*gosnmpd.SubAgent{{
			CommunityIDs: []string{community},
			OIDs:         items,
		}},
	})
	if err := srv.ListenUDP("udp", "127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.ServeForever() }()
	t.Cleanup(srv.Shutdown)
	time.Sleep(20 * time.Millisecond)
	return srv
}

func TestEnrichFromPeer(t *testing.T) {
	t.Parallel()
	srv := startPeer(t, "lab", map[string]string{
		"1.3.6.1.2.1.1.5.0": "peer-host",
		"1.3.6.1.2.1.1.1.0": "Linux router",
	})
	tab := New([]config.Neighbor{{
		Port: 3,
		SNMP: &config.NeighborSNMP{Address: srv.Address().String(), Community: "lab"},
	}})
	tab.Enrich()
	n, ok := tab.Get(3)
	if !ok || n.SysName != "peer-host" || n.SysDesc != "Linux router" || n.ChassisID != "peer-host" || n.PortID != "" || n.ChassisIDSubtype != 7 {
		t.Fatalf("enriched = %+v ok=%v", n, ok)
	}

	// Complete identity must not dial a dead peer.
	tab = New([]config.Neighbor{{
		Port: 1, SysName: "cfg", SysDesc: "desc", PortID: "eth0", ChassisID: "cfg",
		SNMP: &config.NeighborSNMP{Address: "127.0.0.1:1"},
	}})
	tab.Enrich()
	n, _ = tab.Get(1)
	if n.SysName != "cfg" {
		t.Fatalf("complete row mutated = %+v", n)
	}
}
