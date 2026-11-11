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

// A reboot drops counters to 0 while sysUpTime keeps going. Mark that so
// pollers drop the delta (RFC 2863 ifCounterDiscontinuityTime).
func (s *Store) Set(snap *netgear.Snapshot) {
	prev := s.Get()
	for i := range snap.Ports {
		p := &snap.Ports[i]
		var old netgear.Port
		if i < len(prev.Ports) {
			old = prev.Ports[i]
		}
		p.ResetTicks = old.ResetTicks
		if p.RxBytes < old.RxBytes || p.TxBytes < old.TxBytes || p.CRCErrors < old.CRCErrors {
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
