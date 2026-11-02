package agent

import (
	"testing"

	"github.com/bingoohuang/gosnmpd"
	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/netgear"
	"github.com/kgrubb/snare/internal/topology"
)

func TestPethDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status string
		admin  bool
		want   int
	}{
		{name: "disabled", status: "delivering", admin: false, want: 1},
		{name: "delivering", status: "delivering_power", admin: true, want: 3},
		{name: "searching", status: "searching", admin: true, want: 2},
		{name: "empty status", status: "", admin: true, want: 2},
		{name: "fault", status: "fault", admin: true, want: 4},
		{name: "other", status: "idle", admin: true, want: 2},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := pethDetection(tt.status, tt.admin); got != tt.want {
				t.Fatalf("pethDetection = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildOIDs(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Switch: config.Switch{Ports: 2},
		Sys:    config.Sys{Contact: "ops@example.com", Location: "lab", Name: "snare-lab"},
	}
	store := NewStore()
	store.Set(&netgear.Snapshot{
		Model:     "GS308EP",
		Serial:    "ABC",
		Firmware:  "1.0",
		Name:      "from-switch",
		PoEBudget: 62,
		Ports: []netgear.Port{
			{Index: 1, Up: true, SpeedMbps: 1000, RxBytes: 10, TxBytes: 20, CRCErrors: 1, PoEAdmin: true, PoEStatus: "delivering", PoEClass: 4, PoEPowerW: 5},
			{Index: 2, Up: false, PoEAdmin: false, PoEStatus: "", PoEClass: 0},
		},
	})

	topo := topology.New([]config.Neighbor{{
		Port: 1, SysName: "peer-a", PortID: "eth0", ChassisID: "peer-a",
	}})
	items := buildOIDs(store, cfg, topo)
	if len(items) == 0 {
		t.Fatal("expected OID items")
	}

	byOID := map[string]any{}
	for _, it := range items {
		v, err := it.OnGet()
		if err != nil {
			t.Fatalf("OnGet %s: %v", it.OID, err)
		}
		byOID[it.OID] = v
	}

	if _, ok := byOID[oidSysDescr]; !ok {
		t.Fatal("missing sysDescr")
	}
	if _, ok := byOID[oidSysObjectID]; !ok {
		t.Fatal("missing sysObjectID")
	}
	if _, ok := byOID[oidSysContact]; !ok {
		t.Fatal("missing sysContact")
	}
	if _, ok := byOID[oidIfNumber]; !ok {
		t.Fatal("missing ifNumber")
	}

	// ifEntry / ifXEntry columns on port 1
	wantCols := []string{
		oidIfEntry + ".4.1", oidIfEntry + ".6.1", oidIfEntry + ".9.1",
		oidIfEntry + ".11.1", oidIfEntry + ".12.1", oidIfEntry + ".13.1",
		oidIfEntry + ".15.1", oidIfEntry + ".17.1", oidIfEntry + ".18.1",
		oidIfEntry + ".19.1", oidIfEntry + ".20.1", oidIfEntry + ".21.1",
		oidIfXEntry + ".2.1", oidIfXEntry + ".3.1", oidIfXEntry + ".4.1",
		oidIfXEntry + ".5.1", oidIfXEntry + ".7.1", oidIfXEntry + ".8.1",
		oidIfXEntry + ".9.1", oidIfXEntry + ".11.1", oidIfXEntry + ".12.1",
		oidIfXEntry + ".13.1", oidIfXEntry + ".14.1", oidIfXEntry + ".15.1",
		oidIfXEntry + ".16.1", oidIfXEntry + ".17.1", oidIfXEntry + ".18.1",
		oidIfXEntry + ".19.1",
		oidIfTableLastChange, oidIfStackLastChange,
	}
	for _, oid := range wantCols {
		if _, ok := byOID[oid]; !ok {
			t.Fatalf("missing %s", oid)
		}
	}
	if got := gosnmpd.Asn1IntegerUnwrap(byOID[oidIfEntry+".4.1"]); got != 1500 {
		t.Fatalf("ifMtu = %d, want 1500", got)
	}
	if got := gosnmpd.Asn1Gauge32Unwrap(byOID[oidIfXEntry+".15.1"]); got != 1000 {
		t.Fatalf("ifHighSpeed = %d, want 1000", got)
	}
	if got := gosnmpd.Asn1OctetStringUnwrap(byOID[oidLldpRemEntry+".9.0.1.1"]); got != "peer-a" {
		t.Fatalf("lldpRemSysName = %q, want peer-a", got)
	}

	// sysName prefers config
	cfg.Sys.Name = ""
	empty := topology.New(nil)
	items = buildOIDs(store, cfg, empty)
	for _, it := range items {
		if it.OID != oidSysName {
			continue
		}
		v, err := it.OnGet()
		if err != nil {
			t.Fatal(err)
		}
		if v == nil {
			t.Fatal("sysName nil")
		}
	}

	store.Set(&netgear.Snapshot{})
	cfg.Sys.Name = ""
	items = buildOIDs(store, cfg, empty)
	for _, it := range items {
		if it.OID != oidSysName {
			continue
		}
		if _, err := it.OnGet(); err != nil {
			t.Fatal(err)
		}
	}

	// port beyond snapshot length uses empty port
	cfg.Switch.Ports = 3
	store.Set(&netgear.Snapshot{Ports: []netgear.Port{{Index: 1}}})
	items = buildOIDs(store, cfg, empty)
	found := false
	for _, it := range items {
		if it.OID == oidIfEntry+".8.3" {
			found = true
		}
		if _, err := it.OnGet(); err != nil {
			t.Fatalf("%s: %v", it.OID, err)
		}
	}
	if !found {
		t.Fatal("missing ifOperStatus for port 3")
	}
}
