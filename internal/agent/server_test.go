package agent

import (
	"testing"

	"github.com/slayercat/gosnmp"
)

func TestAuthPriv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		auth, priv string
		wantAuth   gosnmp.SnmpV3AuthProtocol
		wantPriv   gosnmp.SnmpV3PrivProtocol
	}{
		{"", "", gosnmp.SHA, gosnmp.AES},
		{"MD5", "DES", gosnmp.MD5, gosnmp.DES},
		{"md5", "des", gosnmp.MD5, gosnmp.DES},
		{"SHA", "AES", gosnmp.SHA, gosnmp.AES},
		{"SHA1", "AES128", gosnmp.SHA, gosnmp.AES},
		{"unknown", "unknown", gosnmp.SHA, gosnmp.AES},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.auth+"_"+tt.priv, func(t *testing.T) {
			t.Parallel()
			a, p := authPriv(tt.auth, tt.priv)
			if a != tt.wantAuth || p != tt.wantPriv {
				t.Fatalf("authPriv(%q,%q) = %v/%v, want %v/%v", tt.auth, tt.priv, a, p, tt.wantAuth, tt.wantPriv)
			}
		})
	}
}
