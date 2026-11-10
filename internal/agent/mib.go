package agent

import (
	"fmt"
	"strings"

	"github.com/bingoohuang/gosnmpd"
	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/netgear"
	"github.com/kgrubb/snare/internal/topology"
	"github.com/slayercat/gosnmp"
)

// OID layout mirrors Netgear Smart Managed Pro agents:
// SNMPv2-MIB, IF-MIB (incl. ifXTable), POWER-ETHERNET-MIB (RFC 3621),
// LLDP-MIB rem (from topology config, not Plus UI).
// sysObjectID uses Netgear enterprise 4526 with a Plus-facade leaf.
const (
	oidSysDescr          = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID       = "1.3.6.1.2.1.1.2.0"
	oidSysUpTime         = "1.3.6.1.2.1.1.3.0"
	oidSysContact        = "1.3.6.1.2.1.1.4.0"
	oidSysName           = "1.3.6.1.2.1.1.5.0"
	oidSysLocation       = "1.3.6.1.2.1.1.6.0"
	oidSysServices       = "1.3.6.1.2.1.1.7.0"
	oidIfNumber          = "1.3.6.1.2.1.2.1.0"
	oidIfEntry           = "1.3.6.1.2.1.2.2.1"
	oidIfXEntry          = "1.3.6.1.2.1.31.1.1.1"
	oidIfTableLastChange = "1.3.6.1.2.1.31.1.5.0"
	oidIfStackLastChange = "1.3.6.1.2.1.31.1.6.0"
	oidPethPseEntry      = "1.3.6.1.2.1.105.1.1.1"
	oidPethMainPse       = "1.3.6.1.2.1.105.1.3.1.1"
	oidLldpLocChassisSub = "1.0.8802.1.1.2.1.3.1.0"
	oidLldpLocChassisID  = "1.0.8802.1.1.2.1.3.2.0"
	oidLldpLocSysName    = "1.0.8802.1.1.2.1.3.3.0"
	oidLldpLocSysDesc    = "1.0.8802.1.1.2.1.3.4.0"
	oidLldpRemEntry      = "1.0.8802.1.1.2.1.4.1.1"
	// enterprises.netgear.snarePlus.gs308ep
	oidNetgearPlus = "1.3.6.1.4.1.4526.100.308"
)

func buildOIDs(store *Store, cfg *config.Config, topo *topology.Table) []*gosnmpd.PDUValueControlItem {
	get := store.Get
	items := []*gosnmpd.PDUValueControlItem{
		strOID(oidSysDescr, func() string {
			s := get()
			return fmt.Sprintf("snare %s %s firmware %s (Netgear Plus SNMP facade)", s.Model, s.Serial, s.Firmware)
		}),
		oidOID(oidSysObjectID, oidNetgearPlus),
		ticksOID(oidSysUpTime, store.UpTimeTicks),
		strOID(oidSysContact, func() string { return cfg.Sys.Contact }),
		strOID(oidSysName, func() string {
			if cfg.Sys.Name != "" {
				return cfg.Sys.Name
			}
			if n := get().Name; n != "" {
				return n
			}
			return "snare"
		}),
		strOID(oidSysLocation, func() string { return cfg.Sys.Location }),
		intOID(oidSysServices, func() int { return 2 }), // data link
		intOID(oidIfNumber, func() int { return len(get().Ports) }),
		ticksOID(oidIfTableLastChange, func() uint32 { return 0 }),
		ticksOID(oidIfStackLastChange, func() uint32 { return 0 }),
		intOID(oidLldpLocChassisSub, func() int { return 7 }), // local
		strOID(oidLldpLocChassisID, func() string {
			if cfg.Sys.Name != "" {
				return cfg.Sys.Name
			}
			return "snare"
		}),
		strOID(oidLldpLocSysName, func() string {
			if cfg.Sys.Name != "" {
				return cfg.Sys.Name
			}
			if n := get().Name; n != "" {
				return n
			}
			return "snare"
		}),
		strOID(oidLldpLocSysDesc, func() string {
			s := get()
			return fmt.Sprintf("snare %s %s firmware %s", s.Model, s.Serial, s.Firmware)
		}),
		gaugeOID(oidPethMainPse+".2.1", func() uint { return get().PoEBudget }),
		intOID(oidPethMainPse+".3.1", func() int { return 1 }), // on
		gaugeOID(oidPethMainPse+".4.1", func() uint {
			var w float64
			for _, p := range get().Ports {
				w += p.PoEPowerW
			}
			return uint(w + 0.5)
		}),
	}

	n := cfg.Switch.Ports
	for i := 1; i <= n; i++ {
		idx := i
		port := func() netgear.Port {
			s := get()
			if idx <= len(s.Ports) {
				return s.Ports[idx-1]
			}
			return netgear.Port{Index: idx}
		}
		upDown := func() int {
			if port().Up {
				return 1
			}
			return 2
		}
		// Every ifEntry/ifXEntry leaf. A hole makes gosnmpd NoSuchName the whole GET.
		// Counters the Plus UI lacks return 0.
		items = append(items,
			intOID(fmt.Sprintf("%s.1.%d", oidIfEntry, idx), func() int { return idx }),
			strOID(fmt.Sprintf("%s.2.%d", oidIfEntry, idx), func() string { return fmt.Sprintf("Port %d", idx) }),
			intOID(fmt.Sprintf("%s.3.%d", oidIfEntry, idx), func() int { return 6 }), // ethernetCsmacd
			intOID(fmt.Sprintf("%s.4.%d", oidIfEntry, idx), func() int { return 1500 }),
			gaugeOID(fmt.Sprintf("%s.5.%d", oidIfEntry, idx), func() uint {
				return uint(port().SpeedMbps * 1_000_000)
			}),
			strOID(fmt.Sprintf("%s.6.%d", oidIfEntry, idx), func() string { return "" }), // no per-port MAC from UI
			intOID(fmt.Sprintf("%s.7.%d", oidIfEntry, idx), upDown),
			intOID(fmt.Sprintf("%s.8.%d", oidIfEntry, idx), upDown),
			ticksOID(fmt.Sprintf("%s.9.%d", oidIfEntry, idx), func() uint32 { return 0 }),
			counterOID(fmt.Sprintf("%s.10.%d", oidIfEntry, idx), func() uint { return uint(port().RxBytes) }),
			counterOID(fmt.Sprintf("%s.11.%d", oidIfEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.12.%d", oidIfEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.13.%d", oidIfEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.14.%d", oidIfEntry, idx), func() uint { return uint(port().CRCErrors) }),
			counterOID(fmt.Sprintf("%s.15.%d", oidIfEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.16.%d", oidIfEntry, idx), func() uint { return uint(port().TxBytes) }),
			counterOID(fmt.Sprintf("%s.17.%d", oidIfEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.18.%d", oidIfEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.19.%d", oidIfEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.20.%d", oidIfEntry, idx), func() uint { return 0 }),
			gaugeOID(fmt.Sprintf("%s.21.%d", oidIfEntry, idx), func() uint { return 0 }),
			strOID(fmt.Sprintf("%s.1.%d", oidIfXEntry, idx), func() string { return fmt.Sprintf("Port %d", idx) }),
			counterOID(fmt.Sprintf("%s.2.%d", oidIfXEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.3.%d", oidIfXEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.4.%d", oidIfXEntry, idx), func() uint { return 0 }),
			counterOID(fmt.Sprintf("%s.5.%d", oidIfXEntry, idx), func() uint { return 0 }),
			counter64OID(fmt.Sprintf("%s.6.%d", oidIfXEntry, idx), func() uint64 { return port().RxBytes }),
			counter64OID(fmt.Sprintf("%s.7.%d", oidIfXEntry, idx), func() uint64 { return 0 }),
			counter64OID(fmt.Sprintf("%s.8.%d", oidIfXEntry, idx), func() uint64 { return 0 }),
			counter64OID(fmt.Sprintf("%s.9.%d", oidIfXEntry, idx), func() uint64 { return 0 }),
			counter64OID(fmt.Sprintf("%s.10.%d", oidIfXEntry, idx), func() uint64 { return port().TxBytes }),
			counter64OID(fmt.Sprintf("%s.11.%d", oidIfXEntry, idx), func() uint64 { return 0 }),
			counter64OID(fmt.Sprintf("%s.12.%d", oidIfXEntry, idx), func() uint64 { return 0 }),
			counter64OID(fmt.Sprintf("%s.13.%d", oidIfXEntry, idx), func() uint64 { return 0 }),
			intOID(fmt.Sprintf("%s.14.%d", oidIfXEntry, idx), func() int { return 1 }), // enabled (we send link traps)
			gaugeOID(fmt.Sprintf("%s.15.%d", oidIfXEntry, idx), func() uint { return uint(port().SpeedMbps) }),
			intOID(fmt.Sprintf("%s.16.%d", oidIfXEntry, idx), func() int { return 2 }), // false
			intOID(fmt.Sprintf("%s.17.%d", oidIfXEntry, idx), func() int { return 1 }), // true
			strOID(fmt.Sprintf("%s.18.%d", oidIfXEntry, idx), func() string { return "" }),
			ticksOID(fmt.Sprintf("%s.19.%d", oidIfXEntry, idx), func() uint32 { return port().ResetTicks }),
			// POWER-ETHERNET-MIB pethPsePortTable index group=1,port=N
			intOID(fmt.Sprintf("%s.3.1.%d", oidPethPseEntry, idx), func() int {
				if port().PoEAdmin {
					return 1
				}
				return 2
			}),
			intOID(fmt.Sprintf("%s.6.1.%d", oidPethPseEntry, idx), func() int {
				return pethDetection(port().PoEStatus, port().PoEAdmin)
			}),
			intOID(fmt.Sprintf("%s.10.1.%d", oidPethPseEntry, idx), func() int {
				if port().PoEClass > 0 {
					return port().PoEClass
				}
				return 1
			}),
		)
	}

	for _, port := range topo.Ports() {
		p := port
		// lldpRemTable index: timeMark=0, localPortNum, remIndex=1
		idx := fmt.Sprintf("0.%d.1", p)
		neighbor := func() config.Neighbor {
			n, _ := topo.Get(p)
			return n
		}
		items = append(items,
			intOID(fmt.Sprintf("%s.4.%s", oidLldpRemEntry, idx), func() int { return neighbor().ChassisIDSubtype }),
			strOID(fmt.Sprintf("%s.5.%s", oidLldpRemEntry, idx), func() string { return neighbor().ChassisID }),
			intOID(fmt.Sprintf("%s.6.%s", oidLldpRemEntry, idx), func() int { return neighbor().PortIDSubtype }),
			strOID(fmt.Sprintf("%s.7.%s", oidLldpRemEntry, idx), func() string { return neighbor().PortID }),
			strOID(fmt.Sprintf("%s.8.%s", oidLldpRemEntry, idx), func() string { return neighbor().PortDesc }),
			strOID(fmt.Sprintf("%s.9.%s", oidLldpRemEntry, idx), func() string { return neighbor().SysName }),
			strOID(fmt.Sprintf("%s.10.%s", oidLldpRemEntry, idx), func() string { return neighbor().SysDesc }),
		)
	}
	return items
}

func pethDetection(status string, admin bool) int {
	// RFC 3621: disabled(1) searching(2) deliveringPower(3) fault(4) test(5) otherFault(6)
	if !admin {
		return 1
	}
	switch {
	case strings.Contains(status, "deliver"):
		return 3
	case strings.Contains(status, "search"), status == "":
		return 2
	case strings.Contains(status, "fault"):
		return 4
	default:
		return 2
	}
}

func strOID(oid string, f func() string) *gosnmpd.PDUValueControlItem {
	return &gosnmpd.PDUValueControlItem{
		OID: oid, Type: gosnmp.OctetString,
		OnGet: func() (any, error) { return gosnmpd.Asn1OctetStringWrap(f()), nil },
	}
}

func oidOID(oid, value string) *gosnmpd.PDUValueControlItem {
	return &gosnmpd.PDUValueControlItem{
		OID: oid, Type: gosnmp.ObjectIdentifier,
		OnGet: func() (any, error) { return gosnmpd.Asn1ObjectIdentifierWrap(value), nil },
	}
}

func intOID(oid string, f func() int) *gosnmpd.PDUValueControlItem {
	return &gosnmpd.PDUValueControlItem{
		OID: oid, Type: gosnmp.Integer,
		OnGet: func() (any, error) { return gosnmpd.Asn1IntegerWrap(f()), nil },
	}
}

func gaugeOID(oid string, f func() uint) *gosnmpd.PDUValueControlItem {
	return &gosnmpd.PDUValueControlItem{
		OID: oid, Type: gosnmp.Gauge32,
		OnGet: func() (any, error) { return gosnmpd.Asn1Gauge32Wrap(f()), nil },
	}
}

func counterOID(oid string, f func() uint) *gosnmpd.PDUValueControlItem {
	return &gosnmpd.PDUValueControlItem{
		OID: oid, Type: gosnmp.Counter32,
		OnGet: func() (any, error) { return gosnmpd.Asn1Counter32Wrap(f()), nil },
	}
}

func counter64OID(oid string, f func() uint64) *gosnmpd.PDUValueControlItem {
	return &gosnmpd.PDUValueControlItem{
		OID: oid, Type: gosnmp.Counter64,
		OnGet: func() (any, error) { return gosnmpd.Asn1Counter64Wrap(f()), nil },
	}
}

func ticksOID(oid string, f func() uint32) *gosnmpd.PDUValueControlItem {
	return &gosnmpd.PDUValueControlItem{
		OID: oid, Type: gosnmp.TimeTicks,
		OnGet: func() (any, error) { return gosnmpd.Asn1TimeTicksWrap(f()), nil },
	}
}
