package agent

import (
	"testing"
	"time"

	"github.com/kgrubb/snare/internal/netgear"
)

func TestStoreGetSetAndUptime(t *testing.T) {
	t.Parallel()

	s := NewStore()
	if s.Fresh(time.Hour) {
		t.Fatal("no scrape yet is not fresh")
	}
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

	if s.Fresh(time.Minute) {
		t.Fatal("a snapshot without a fetch time is not fresh")
	}
	s.Set(&netgear.Snapshot{FetchedAt: time.Now()})
	if !s.Fresh(time.Minute) {
		t.Fatal("a just-fetched snapshot is fresh")
	}

	time.Sleep(15 * time.Millisecond)
	if s.UpTimeTicks() == 0 {
		t.Fatal("uptime ticks should advance")
	}
}

func TestStoreResetTicks(t *testing.T) {
	t.Parallel()

	s := NewStore()
	set := func(rx, tx uint64) {
		s.Set(&netgear.Snapshot{Ports: []netgear.Port{{Index: 1, RxBytes: rx, TxBytes: tx}}})
	}

	set(1000, 2000)
	if got := s.Get().Ports[0].ResetTicks; got != 0 {
		t.Fatalf("first scrape discontinuity = %d, want 0", got)
	}

	time.Sleep(15 * time.Millisecond)
	set(0xFFFFFFFF<<32|4288590000, 2500)
	if p := s.Get().Ports[0]; p.RxBytes != 1000 || p.TxBytes != 2500 {
		t.Fatalf("negative reading served rx=%d tx=%d, want 1000 2500", p.RxBytes, p.TxBytes)
	}

	set(10, 2600)
	disc := s.Get().Ports[0].ResetTicks
	if disc == 0 || disc > s.UpTimeTicks() {
		t.Fatalf("reset discontinuity = %d, want in (0, %d]", disc, s.UpTimeTicks())
	}

	time.Sleep(15 * time.Millisecond)
	set(20, 2700)
	if got := s.Get().Ports[0].ResetTicks; got != disc {
		t.Fatalf("discontinuity moved to %d on a normal scrape, want %d", got, disc)
	}
}
