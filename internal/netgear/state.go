package netgear

import "time"

type Snapshot struct {
	FetchedAt time.Time
	Model     string
	Name      string
	Serial    string
	Firmware  string
	Ports     []Port
	PoEBudget uint
}

type Port struct {
	Index      int
	Up         bool
	SpeedMbps  uint64
	RxBytes    uint64
	TxBytes    uint64
	CRCErrors  uint64
	PoEAdmin   bool
	PoEStatus  string
	PoEClass   int
	PoEPowerW  float64
	PoEVoltage int
	PoECurrent int
}
