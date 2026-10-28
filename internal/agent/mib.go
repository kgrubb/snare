package agent

import (
	"fmt"
	"strings"

	"github.com/bingoohuang/gosnmpd"
	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/netgear"
	"github.com/slayercat/gosnmp"
)

// OID layout mirrors Netgear Smart Managed Pro agents:
// SNMPv2-MIB, IF-MIB (incl. ifXTable), POWER-ETHERNET-MIB (RFC 3621).
// sysObjectID uses Netgear enterprise 4526 with a Plus-facade leaf.
const (
	oidSysDescr     = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID  = "1.3.6.1.2.1.1.2.0"
	oidSysUpTime    = "1.3.6.1.2.1.1.3.0"
	oidSysContact   = "1.3.6.1.2.1.1.4.0"
	oidSysName      = "1.3.6.1.2.1.1.5.0"
	oidSysLocation  = "1.3.6.1.2.1.1.6.0"
	oidSysServices  = "1.3.6.1.2.1.1.7.0"
	oidIfNumber     = "1.3.6.1.2.1.2.1.0"
	oidIfEntry      = "1.3.6.1.2.1.2.2.1"
	oidIfXEntry     = "1.3.6.1.2.1.31.1.1.1"
	oidPethPseEntry = "1.3.6.1.2.1.105.1.1.1"
	oidPethMainPse  = "1.3.6.1.2.1.105.1.3.1.1"
	// enterprises.netgear.snarePlus.gs308ep
	oidNetgearPlus = "1.3.6.1.4.1.4526.100.308"
)

func buildOIDs(store *Store, cfg *config.Config) []*gosnmpd.PDUValueControlItem {
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
		items = append(items,
			intOID(fmt.Sprintf("%s.1.%d", oidIfEntry, idx), func() int { return idx }),
			strOID(fmt.Sprintf("%s.2.%d", oidIfEntry, idx), func() string { return fmt.Sprintf("Port %d", idx) }),
			intOID(fmt.Sprintf("%s.3.%d", oidIfEntry, idx), func() int { return 6 }), // ethernetCsmacd
			gaugeOID(fmt.Sprintf("%s.5.%d", oidIfEntry, idx), func() uint {
				return uint(port().SpeedMbps * 1_000_000)
			}),
			intOID(fmt.Sprintf("%s.7.%d", oidIfEntry, idx), func() int {
				if port().Up {
					return 1
				}
				return 2
			}),
			intOID(fmt.Sprintf("%s.8.%d", oidIfEntry, idx), func() int {
				if port().Up {
					return 1
				}
				return 2
			}),
			counterOID(fmt.Sprintf("%s.10.%d", oidIfEntry, idx), func() uint { return uint(port().RxBytes) }),
			counterOID(fmt.Sprintf("%s.16.%d", oidIfEntry, idx), func() uint { return uint(port().TxBytes) }),
			counterOID(fmt.Sprintf("%s.14.%d", oidIfEntry, idx), func() uint { return uint(port().CRCErrors) }),
			counter64OID(fmt.Sprintf("%s.6.%d", oidIfXEntry, idx), func() uint64 { return port().RxBytes }),
			counter64OID(fmt.Sprintf("%s.10.%d", oidIfXEntry, idx), func() uint64 { return port().TxBytes }),
			strOID(fmt.Sprintf("%s.1.%d", oidIfXEntry, idx), func() string { return fmt.Sprintf("Port %d", idx) }),
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
