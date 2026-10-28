package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		wantErr string
		check   func(t *testing.T, c *Config)
	}{
		{
			name: "defaults and overrides",
			yaml: `
switch:
  url: http://switch.local
  password: secret
sys:
  name: gs308ep
`,
			check: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Listen != ":161" {
					t.Fatalf("Listen = %q, want :161", c.Listen)
				}
				if c.Community != "public" {
					t.Fatalf("Community = %q, want public", c.Community)
				}
				if c.Poll != 30*time.Second {
					t.Fatalf("Poll = %v, want 30s", c.Poll)
				}
				if c.Switch.Ports != 8 {
					t.Fatalf("Ports = %d, want 8", c.Switch.Ports)
				}
				if c.Switch.PoEBudgetWatts != 62 {
					t.Fatalf("PoEBudgetWatts = %d, want 62", c.Switch.PoEBudgetWatts)
				}
				if c.SNMPv3.AuthProto != "SHA" || c.SNMPv3.PrivProto != "AES" {
					t.Fatalf("SNMPv3 protos = %s/%s", c.SNMPv3.AuthProto, c.SNMPv3.PrivProto)
				}
				if c.Sys.Name != "gs308ep" {
					t.Fatalf("Sys.Name = %q", c.Sys.Name)
				}
			},
		},
		{
			name: "explicit values",
			yaml: `
listen: ":1161"
community: lab
poll: 5s
switch:
  url: http://192.168.1.1
  password: pw
  ports: 16
  poe_budget_watts: 120
snmpv3:
  username: admin
  auth_proto: MD5
  priv_proto: DES
traps:
  enabled: true
  targets: ["192.168.1.10:162"]
`,
			check: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Listen != ":1161" || c.Community != "lab" || c.Poll != 5*time.Second {
					t.Fatalf("listen/community/poll = %q %q %v", c.Listen, c.Community, c.Poll)
				}
				if c.Switch.Ports != 16 || c.Switch.PoEBudgetWatts != 120 {
					t.Fatalf("switch ports/budget = %d/%d", c.Switch.Ports, c.Switch.PoEBudgetWatts)
				}
				if c.SNMPv3.Username != "admin" || c.SNMPv3.AuthProto != "MD5" || c.SNMPv3.PrivProto != "DES" {
					t.Fatalf("snmpv3 = %+v", c.SNMPv3)
				}
				if !c.Traps.Enabled || len(c.Traps.Targets) != 1 {
					t.Fatalf("traps = %+v", c.Traps)
				}
			},
		},
		{
			name: "nonpositive ports and poll reset",
			yaml: `
poll: 0s
switch:
  url: http://x
  password: y
  ports: 0
`,
			check: func(t *testing.T, c *Config) {
				t.Helper()
				if c.Switch.Ports != 8 {
					t.Fatalf("Ports = %d, want 8", c.Switch.Ports)
				}
				if c.Poll != 30*time.Second {
					t.Fatalf("Poll = %v, want 30s", c.Poll)
				}
			},
		},
		{
			name:    "missing url",
			yaml:    "switch:\n  password: x\n",
			wantErr: "switch.url and switch.password are required",
		},
		{
			name:    "missing password",
			yaml:    "switch:\n  url: http://x\n",
			wantErr: "switch.url and switch.password are required",
		},
		{
			name:    "invalid yaml",
			yaml:    ":\n  - bad",
			wantErr: "yaml",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			c, err := Load(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.wantErr == "yaml" {
					return
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("err = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			tt.check(t, c)
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
