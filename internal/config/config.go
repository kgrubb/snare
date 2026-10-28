package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen    string        `yaml:"listen"`
	Community string        `yaml:"community"`
	Poll      time.Duration `yaml:"poll"`
	Switch    Switch        `yaml:"switch"`
	SNMPv3    SNMPv3        `yaml:"snmpv3"`
	Traps     Traps         `yaml:"traps"`
	Sys       Sys           `yaml:"sys"`
}

type Switch struct {
	URL            string `yaml:"url"`
	Password       string `yaml:"password"`
	Ports          int    `yaml:"ports"`
	PoEBudgetWatts uint   `yaml:"poe_budget_watts"`
}

type SNMPv3 struct {
	Username  string `yaml:"username"`
	AuthPass  string `yaml:"auth_pass"`
	PrivPass  string `yaml:"priv_pass"`
	AuthProto string `yaml:"auth_proto"` // MD5, SHA
	PrivProto string `yaml:"priv_proto"` // DES, AES
}

type Traps struct {
	Enabled bool     `yaml:"enabled"`
	Targets []string `yaml:"targets"` // host:port
}

type Sys struct {
	Contact  string `yaml:"contact"`
	Location string `yaml:"location"`
	Name     string `yaml:"name"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := &Config{
		Listen:    ":161",
		Community: "public",
		Poll:      30 * time.Second,
		Switch: Switch{
			Ports:          8,
			PoEBudgetWatts: 62,
		},
		SNMPv3: SNMPv3{
			AuthProto: "SHA",
			PrivProto: "AES",
		},
	}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	if c.Switch.URL == "" || c.Switch.Password == "" {
		return nil, fmt.Errorf("switch.url and switch.password are required")
	}
	if c.Switch.Ports <= 0 {
		c.Switch.Ports = 8
	}
	if c.Poll <= 0 {
		c.Poll = 30 * time.Second
	}
	return c, nil
}
