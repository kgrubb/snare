package agent

import (
	"testing"
	"time"

	"github.com/kgrubb/snare/internal/netgear"
)

func TestStoreGetSetAndUptime(t *testing.T) {
	t.Parallel()

	s := NewStore()
	empty := s.Get()
	if empty == nil || len(empty.Ports) != 0 {
		t.Fatalf("empty get = %+v", empty)
	}

	snap := &netgear.Snapshot{Model: "GS308EP", Ports: []netgear.Port{{Index: 1, Up: true}}}
	s.Set(snap)
	got := s.Get()
	if got.Model != "GS308EP" || len(got.Ports) != 1 || !got.Ports[0].Up {
		t.Fatalf("get = %+v", got)
	}

	time.Sleep(15 * time.Millisecond)
	if s.UpTimeTicks() == 0 {
		t.Fatal("uptime ticks should advance")
	}
}
