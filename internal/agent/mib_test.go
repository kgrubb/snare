package agent

import (
	"testing"

	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/netgear"
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

	items := buildOIDs(store, cfg)
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

	// sysName prefers config
	cfg.Sys.Name = ""
	items = buildOIDs(store, cfg)
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
	items = buildOIDs(store, cfg)
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
	items = buildOIDs(store, cfg)
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
