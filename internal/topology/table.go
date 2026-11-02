package topology

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/kgrubb/snare/internal/config"
)

// Table is declared LLDP rem rows, optionally enriched from peer SNMP.
type Table struct {
	mu   sync.RWMutex
	rows map[int]config.Neighbor
}

func New(cfgNeighbors []config.Neighbor) *Table {
	t := &Table{rows: map[int]config.Neighbor{}}
	for _, n := range cfgNeighbors {
		if n.ChassisIDSubtype == 0 {
			n.ChassisIDSubtype = 7 // local (set 4 explicitly for a MAC chassis_id)
		}
		if n.PortIDSubtype == 0 {
			n.PortIDSubtype = 5 // interfaceName
		}
		t.rows[n.Port] = n
	}
	return t
}

func (t *Table) Ports() []int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]int, 0, len(t.rows))
	for p := range t.rows {
		out = append(out, p)
	}
	return out
}

func (t *Table) Get(port int) (config.Neighbor, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n, ok := t.rows[port]
	return n, ok
}

// Enrich updates empty identity fields from each neighbor's peer SNMP agent.
func (t *Table) Enrich() {
	t.mu.RLock()
	snap := make(map[int]config.Neighbor, len(t.rows))
	for p, n := range t.rows {
		snap[p] = n
	}
	t.mu.RUnlock()

	for p, n := range snap {
		if n.SNMP == nil || n.SNMP.Address == "" {
			continue
		}
		if n.SysName != "" && n.ChassisID != "" && n.SysDesc != "" {
			continue
		}
		if enrichOne(&n) != nil {
			continue
		}
		t.mu.Lock()
		t.rows[p] = n
		t.mu.Unlock()
	}
}

func enrichOne(n *config.Neighbor) error {
	host, port := splitAddr(n.SNMP.Address)
	community := n.SNMP.Community
	if community == "" {
		community = "public"
	}
	client := &gosnmp.GoSNMP{
		Target:    host,
		Port:      port,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   3 * time.Second,
		Retries:   1,
	}
	if err := client.Connect(); err != nil {
		return err
	}
	defer func() { _ = client.Conn.Close() }()

	// sysName + sysDescr only. ifDescr.1 is usually lo on Linux peers.
	pkt, err := client.Get([]string{
		"1.3.6.1.2.1.1.5.0",
		"1.3.6.1.2.1.1.1.0",
	})
	if err != nil {
		return err
	}
	for _, v := range pkt.Variables {
		val := snmpString(v.Value)
		if val == "" {
			continue
		}
		switch strings.TrimPrefix(v.Name, ".") {
		case "1.3.6.1.2.1.1.5.0":
			if n.SysName == "" {
				n.SysName = val
			}
			if n.ChassisID == "" {
				n.ChassisID = val
				n.ChassisIDSubtype = 7
			}
		case "1.3.6.1.2.1.1.1.0":
			if n.SysDesc == "" {
				n.SysDesc = val
			}
		}
	}
	if n.ChassisID == "" && n.SysName != "" {
		n.ChassisID = n.SysName
		n.ChassisIDSubtype = 7
	}
	return nil
}

func snmpString(v any) string {
	switch t := v.(type) {
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return ""
	}
}

func splitAddr(addr string) (string, uint16) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 161
	}
	p, err := strconv.Atoi(portStr)
	if err != nil || p <= 0 || p > 65535 {
		return host, 161
	}
	return host, uint16(p)
}
