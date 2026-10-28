package trap

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/kgrubb/snare/internal/agent"
	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/netgear"
)

func trapOperStatus(t *testing.T, raw []byte) int {
	t.Helper()
	pkt, err := (&gosnmp.GoSNMP{}).SnmpDecodePacket(raw)
	if err != nil {
		t.Fatalf("decode trap: %v", err)
	}
	for _, v := range pkt.Variables {
		if strings.Contains(v.Name, "1.3.6.1.2.1.2.2.1.8.") {
			n, ok := v.Value.(int)
			if !ok {
				t.Fatalf("ifOperStatus type %T", v.Value)
			}
			return n
		}
	}
	t.Fatal("ifOperStatus missing")
	return 0
}

func TestOperStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		up   bool
		want int
	}{
		{true, 1},
		{false, 2},
	}
	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			t.Parallel()
			if got := operStatus(tt.up); got != tt.want {
				t.Fatalf("operStatus(%v) = %d, want %d", tt.up, got, tt.want)
			}
		})
	}
}

func TestSplitHostPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		host    string
		port    uint16
		wantErr bool
	}{
		{in: "192.168.1.10:162", host: "192.168.1.10", port: 162},
		{in: "nms.local", host: "nms.local", port: 162},
		{in: "127.0.0.1:9999", host: "127.0.0.1", port: 9999},
		{in: "::1", wantErr: true}, // ambiguous without brackets
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			h, p, err := splitHostPort(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if h != tt.host || p != tt.port {
				t.Fatalf("got %s:%d, want %s:%d", h, p, tt.host, tt.port)
			}
		})
	}
}

func TestDiffAndSend(t *testing.T) {
	t.Parallel()

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{Traps: config.Traps{Enabled: false, Targets: []string{"127.0.0.1:162"}}}
		w := New(cfg, agent.NewStore())
		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 1, Up: true}}})
		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 1, Up: false}}})
	})

	t.Run("no targets", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{Traps: config.Traps{Enabled: true}}
		w := New(cfg, agent.NewStore())
		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 1, Up: true}}})
	})

	t.Run("first sighting no trap", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			Community: "public",
			Traps:     config.Traps{Enabled: true, Targets: []string{"127.0.0.1:1"}},
		}
		w := New(cfg, agent.NewStore())
		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 1, Up: true}}})
		if !w.prev[1] {
			t.Fatal("expected prev recorded")
		}
	})

	t.Run("link flip sends trap", func(t *testing.T) {
		t.Parallel()
		pc, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = pc.Close() })

		addr := pc.LocalAddr().(*net.UDPAddr)
		cfg := &config.Config{
			Community: "public",
			Traps: config.Traps{
				Enabled: true,
				Targets: []string{addr.String()},
			},
		}
		store := agent.NewStore()
		w := New(cfg, store)
		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 2, Up: false}}})
		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 2, Up: true}}})

		_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 2048)
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			t.Fatalf("expected trap datagram: %v", err)
		}
		if got := trapOperStatus(t, buf[:n]); got != 1 {
			t.Fatalf("linkUp ifOperStatus = %d, want 1", got)
		}

		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 2, Up: false}}})
		_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err = pc.ReadFrom(buf)
		if err != nil {
			t.Fatalf("expected linkDown trap: %v", err)
		}
		if got := trapOperStatus(t, buf[:n]); got != 2 {
			t.Fatalf("linkDown ifOperStatus = %d, want 2", got)
		}
	})

	t.Run("bad target keeps prev for retry", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			Community: "public",
			Traps:     config.Traps{Enabled: true, Targets: []string{"::1"}},
		}
		w := New(cfg, agent.NewStore())
		w.prev[1] = true
		w.DiffAndSend(&netgear.Snapshot{Ports: []netgear.Port{{Index: 1, Up: false}}})
		if !w.prev[1] {
			t.Fatal("prev must stay true when send fails so the trap can retry")
		}
	})
}
