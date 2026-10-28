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

func (s *Store) Set(snap *netgear.Snapshot) { s.cur.Store(snap) }

func (s *Store) Get() *netgear.Snapshot {
	if v := s.cur.Load(); v != nil {
		return v
	}
	return &netgear.Snapshot{}
}

func (s *Store) UpTimeTicks() uint32 {
	return uint32(time.Since(s.started).Milliseconds() / 10)
}
