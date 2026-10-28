package agent

import (
	"fmt"
	"strings"

	"github.com/bingoohuang/gosnmpd"
	"github.com/kgrubb/snare/internal/config"
	"github.com/slayercat/gosnmp"
)

func ListenAndServe(cfg *config.Config, store *Store) error {
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

	master := gosnmpd.MasterAgent{
		SecurityConfig: gosnmpd.SecurityConfig{
			AuthoritativeEngineBoots: 1,
			Users:                    users,
		},
		SubAgents: []*gosnmpd.SubAgent{{
			CommunityIDs: []string{cfg.Community},
			OIDs:         buildOIDs(store, cfg),
		}},
	}
	srv := gosnmpd.NewSNMPServer(master)
	if err := srv.ListenUDP("udp", cfg.Listen); err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Listen, err)
	}
	return srv.ServeForever()
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
