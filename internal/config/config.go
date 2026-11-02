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
	Topology  Topology      `yaml:"topology"`
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

// Topology publishes LLDP-MIB rem rows. Plus UI has no LLDP/FDB, so neighbors
// are declared here. Optional snmp fills blank identity from a peer agent.
type Topology struct {
	Neighbors []Neighbor `yaml:"neighbors"`
}

type Neighbor struct {
	Port             int           `yaml:"port"`
	ChassisID        string        `yaml:"chassis_id"`
	ChassisIDSubtype int           `yaml:"chassis_id_subtype"`
	PortID           string        `yaml:"port_id"`
	PortIDSubtype    int           `yaml:"port_id_subtype"`
	PortDesc         string        `yaml:"port_desc"`
	SysName          string        `yaml:"sys_name"`
	SysDesc          string        `yaml:"sys_desc"`
	SNMP             *NeighborSNMP `yaml:"snmp"`
}

type NeighborSNMP struct {
	Address   string `yaml:"address"` // host:port
	Community string `yaml:"community"`
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
	seen := map[int]bool{}
	for i, n := range c.Topology.Neighbors {
		if n.Port < 1 || n.Port > c.Switch.Ports {
			return nil, fmt.Errorf("topology.neighbors[%d].port must be 1..%d", i, c.Switch.Ports)
		}
		if seen[n.Port] {
			return nil, fmt.Errorf("topology.neighbors[%d].port %d duplicated", i, n.Port)
		}
		seen[n.Port] = true
		if n.SysName == "" && n.ChassisID == "" && (n.SNMP == nil || n.SNMP.Address == "") {
			return nil, fmt.Errorf("topology.neighbors[%d] needs sys_name, chassis_id, or snmp.address", i)
		}
	}
	return c, nil
}
