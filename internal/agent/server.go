package agent

import (
	"fmt"
	"net"
	"strings"

	"github.com/bingoohuang/gosnmpd"
	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/topology"
	"github.com/slayercat/gosnmp"
)

func ListenAndServe(cfg *config.Config, store *Store, topo *topology.Table) error {
	users := []gosnmp.UsmSecurityParameters{}
	if cfg.SNMPv3.Username != "" {
		auth, priv := authPriv(cfg.SNMPv3.AuthProto, cfg.SNMPv3.PrivProto)
		users = append(users, gosnmp.UsmSecurityParameters{
			UserName:                 cfg.SNMPv3.Username,
			AuthenticationProtocol:   auth,
			PrivacyProtocol:          priv,
			AuthenticationPassphrase: cfg.SNMPv3.AuthPass,
			PrivacyPassphrase:        cfg.SNMPv3.PrivPass,
		})
	}

	master := &gosnmpd.MasterAgent{
		SecurityConfig: gosnmpd.SecurityConfig{
			AuthoritativeEngineBoots: 1,
			Users:                    users,
		},
		SubAgents: []*gosnmpd.SubAgent{{
			CommunityIDs: []string{cfg.Community},
			OIDs:         buildOIDs(store, cfg, topo),
		}},
	}
	if err := master.ReadyForWork(); err != nil {
		return fmt.Errorf("snmp agent: %w", err)
	}

	pc, err := net.ListenPacket("udp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Listen, err)
	}
	defer func() { _ = pc.Close() }()
	// Stale or missing data would read as zero counters, so stay silent instead.
	return servePacketConn(pc, master, func() bool { return store.Fresh(3 * cfg.Poll) })
}

// servePacketConn answers each datagram in its own goroutine. gosnmpd.ServeForever
// is serial, so pipelined NMS walks retry and then fail on Request ID mismatch.
func servePacketConn(pc net.PacketConn, master *gosnmpd.MasterAgent, ready func() bool) error {
	// shortcut: caps concurrent replies at 64, raise if an NMS pipelines harder
	sem := make(chan struct{}, 64)
	buf := make([]byte, 65535)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return err
		}
		if !ready() {
			continue
		}
		pkt := append([]byte(nil), buf[:n]...)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			out, _ := master.ResponseForBuffer(pkt)
			if len(out) == 0 {
				return
			}
			_, _ = pc.WriteTo(out, addr)
		}()
	}
}

func authPriv(auth, priv string) (gosnmp.SnmpV3AuthProtocol, gosnmp.SnmpV3PrivProtocol) {
	a := gosnmp.SHA
	p := gosnmp.AES
	switch strings.ToUpper(auth) {
	case "MD5":
		a = gosnmp.MD5
	case "SHA", "SHA1":
		a = gosnmp.SHA
	}
	switch strings.ToUpper(priv) {
	case "DES":
		p = gosnmp.DES
	case "AES", "AES128":
		p = gosnmp.AES
	}
	return a, p
}
