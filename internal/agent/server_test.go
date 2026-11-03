package agent

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bingoohuang/gosnmpd"
	"github.com/gosnmp/gosnmp"
	slayer "github.com/slayercat/gosnmp"
)

func TestAuthPriv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		auth, priv string
		wantAuth   slayer.SnmpV3AuthProtocol
		wantPriv   slayer.SnmpV3PrivProtocol
	}{
		{"", "", slayer.SHA, slayer.AES},
		{"MD5", "DES", slayer.MD5, slayer.DES},
		{"md5", "des", slayer.MD5, slayer.DES},
		{"SHA", "AES", slayer.SHA, slayer.AES},
		{"SHA1", "AES128", slayer.SHA, slayer.AES},
		{"unknown", "unknown", slayer.SHA, slayer.AES},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.auth+"_"+tt.priv, func(t *testing.T) {
			t.Parallel()
			a, p := authPriv(tt.auth, tt.priv)
			if a != tt.wantAuth || p != tt.wantPriv {
				t.Fatalf("authPriv(%q,%q) = %v/%v, want %v/%v", tt.auth, tt.priv, a, p, tt.wantAuth, tt.wantPriv)
			}
		})
	}
}

// gosnmpd.ServeForever is serial. Pipelined NMS walks need overlapping handlers.
func TestConcurrentGets(t *testing.T) {
	t.Parallel()

	var inflight, maxInflight atomic.Int32
	master := &gosnmpd.MasterAgent{
		SecurityConfig: gosnmpd.SecurityConfig{
			AuthoritativeEngineBoots: 1,
			AuthoritativeEngineID:    gosnmpd.SNMPEngineID{EngineIDData: "snare-test"},
		},
		SubAgents: []*gosnmpd.SubAgent{{
			CommunityIDs: []string{"public"},
			OIDs: []*gosnmpd.PDUValueControlItem{{
				OID:  "1.3.6.1.2.1.1.5.0",
				Type: slayer.OctetString,
				OnGet: func() (any, error) {
					n := inflight.Add(1)
					for {
						old := maxInflight.Load()
						if n <= old || maxInflight.CompareAndSwap(old, n) {
							break
						}
					}
					time.Sleep(40 * time.Millisecond)
					inflight.Add(-1)
					return gosnmpd.Asn1OctetStringWrap("gs308ep"), nil
				},
			}},
		}},
	}
	if err := master.ReadyForWork(); err != nil {
		t.Fatal(err)
	}

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	go func() { _ = servePacketConn(pc, master) }()

	host, port := mustHostPort(t, pc.LocalAddr().String())
	const n = 12
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &gosnmp.GoSNMP{
				Target: host, Port: port, Community: "public",
				Version: gosnmp.Version2c, Timeout: 2 * time.Second, Retries: 0,
			}
			if err := c.Connect(); err != nil {
				errCh <- err
				return
			}
			defer func() { _ = c.Conn.Close() }()
			pkt, err := c.Get([]string{"1.3.6.1.2.1.1.5.0"})
			if err != nil {
				errCh <- err
				return
			}
			if got := string(pkt.Variables[0].Value.([]byte)); got != "gs308ep" {
				errCh <- fmt.Errorf("sysName=%q", got)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if maxInflight.Load() < 2 {
		t.Fatalf("max inflight handlers = %d, want concurrent serve", maxInflight.Load())
	}
}

func mustHostPort(t *testing.T, addr string) (string, uint16) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return host, uint16(p)
}
