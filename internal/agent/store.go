package agent

import (
	"sync/atomic"
	"time"

	"github.com/kgrubb/snare/internal/netgear"
)

type Store struct {
	started time.Time
	cur     atomic.Pointer[netgear.Snapshot]
}

func NewStore() *Store {
	return &Store{started: time.Now()}
}

// GS3xx firmware sends a negative reading right after a reboot, so keep the last value.
func (s *Store) Set(snap *netgear.Snapshot) {
	prev := s.Get()
	for i := range snap.Ports {
		p := &snap.Ports[i]
		var old netgear.Port
		if i < len(prev.Ports) {
			old = prev.Ports[i]
		}
		p.ResetTicks = old.ResetTicks
		reset := false
		check := func(cur *uint64, last uint64) {
			if int64(*cur) < 0 {
				*cur = last
			} else if *cur < last {
				reset = true
			}
		}
		check(&p.RxBytes, old.RxBytes)
		check(&p.TxBytes, old.TxBytes)
		check(&p.CRCErrors, old.CRCErrors)
		if reset {
			p.ResetTicks = s.UpTimeTicks()
		}
	}
	s.cur.Store(snap)
}

func (s *Store) Get() *netgear.Snapshot {
	if v := s.cur.Load(); v != nil {
		return v
	}
	return &netgear.Snapshot{}
}

// Fresh reports whether a scrape has landed within maxAge.
func (s *Store) Fresh(maxAge time.Duration) bool {
	v := s.cur.Load()
	return v != nil && time.Since(v.FetchedAt) <= maxAge
}

func (s *Store) UpTimeTicks() uint32 {
	return uint32(time.Since(s.started).Milliseconds() / 10)
}
