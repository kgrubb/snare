package trap

import (
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/kgrubb/snare/internal/agent"
	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/netgear"
)

// Watcher diffs port oper status and emits SNMPv2 linkUp/linkDown traps.
type Watcher struct {
	cfg   *config.Config
	store *agent.Store
	prev  map[int]bool
}

func New(cfg *config.Config, store *agent.Store) *Watcher {
	return &Watcher{cfg: cfg, store: store, prev: map[int]bool{}}
}

func (w *Watcher) DiffAndSend(snap *netgear.Snapshot) {
	if !w.cfg.Traps.Enabled || len(w.cfg.Traps.Targets) == 0 {
		return
	}
	for _, p := range snap.Ports {
		was, seen := w.prev[p.Index]
		w.prev[p.Index] = p.Up
		if !seen || was == p.Up {
			continue
		}
		trapOID := "1.3.6.1.6.3.1.1.5.3" // linkDown
		if p.Up {
			trapOID = "1.3.6.1.6.3.1.1.5.4" // linkUp
		}
		if err := w.send(trapOID, p.Index); err != nil {
			log.Printf("trap port %d: %v", p.Index, err)
		}
	}
}

func (w *Watcher) send(trapOID string, ifIndex int) error {
	pdu := []gosnmp.SnmpPDU{
		{Name: "1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: w.store.UpTimeTicks()},
		{Name: "1.3.6.1.6.3.1.1.4.1.0", Type: gosnmp.ObjectIdentifier, Value: trapOID},
		{Name: "1.3.6.1.2.1.2.2.1.1." + strconv.Itoa(ifIndex), Type: gosnmp.Integer, Value: ifIndex},
		{Name: "1.3.6.1.2.1.2.2.1.7." + strconv.Itoa(ifIndex), Type: gosnmp.Integer, Value: operStatus(w.prev[ifIndex])},
		{Name: "1.3.6.1.2.1.2.2.1.8." + strconv.Itoa(ifIndex), Type: gosnmp.Integer, Value: operStatus(w.prev[ifIndex])},
	}
	for _, target := range w.cfg.Traps.Targets {
		host, port, err := splitHostPort(target)
		if err != nil {
			return err
		}
		g := &gosnmp.GoSNMP{
			Target:    host,
			Port:      port,
			Version:   gosnmp.Version2c,
			Community: w.cfg.Community,
			Timeout:   3 * time.Second,
			Retries:   1,
		}
		if err := g.Connect(); err != nil {
			return err
		}
		_, err = g.SendTrap(gosnmp.SnmpTrap{Variables: pdu})
		_ = g.Conn.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func operStatus(up bool) int {
	if up {
		return 1
	}
	return 2
}

func splitHostPort(s string) (string, uint16, error) {
	if !strings.Contains(s, ":") {
		s += ":162"
	}
	h, p, err := net.SplitHostPort(s)
	if err != nil {
		return "", 0, err
	}
	n, err := strconv.ParseUint(p, 10, 16)
	return h, uint16(n), err
}
