package netgear

import (
	"strings"
	"testing"
)

func TestParseRand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		rand string
	}{
		{
			name: "id then value",
			body: `<input id="rand" value="r1">`,
			rand: "r1",
		},
		{
			name: "value then id",
			body: `<input value="r2" id="rand">`,
			rand: "r2",
		},
		{
			name: "missing",
			body: `<html></html>`,
			rand: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseRand(tt.body); got != tt.rand {
				t.Fatalf("parseRand = %q, want %q", got, tt.rand)
			}
		})
	}
}

func TestParseTitleModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "present", body: `<html><head><title> GS308EP </title></head></html>`, want: "GS308EP"},
		{name: "missing", body: `<html><body>no title</body></html>`, want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseTitleModel(tt.body); got != tt.want {
				t.Fatalf("parseTitleModel = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMetadata(t *testing.T) {
	t.Parallel()

	body := `
<span id="switch_name">lab-sw</span>
<span>ml198</span></div><div class="x"><span>SN123</span>
<span>ml089</span></div><div class="x"><span>V1.2.3</span>
`
	name, serial, fw := parseMetadata(body)
	if name != "lab-sw" || serial != "SN123" || fw != "V1.2.3" {
		t.Fatalf("got name=%q serial=%q fw=%q", name, serial, fw)
	}

	name, serial, fw = parseMetadata(`<html></html>`)
	if name != "" || serial != "" || fw != "" {
		t.Fatalf("empty body got name=%q serial=%q fw=%q", name, serial, fw)
	}
}

func TestParseSpeedMbps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want uint64
	}{
		{"1000 Full", 1000},
		{"1G", 1000},
		{"100 Full Duplex", 100},
		{"10 Half", 10},
		{"", 0},
		{"0", 0},
		{"Down", 0},
		{"2500Mbps", 2500},
	}

	for _, tt := range tests {
		tt := tt
		name := tt.in
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := parseSpeedMbps(tt.in); got != tt.want {
				t.Fatalf("parseSpeedMbps(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestParsePortStatus(t *testing.T) {
	t.Parallel()

	body := `
<input name="isShowPot1"><span>Up</span></
<input name="isShowPot2"><span>Down</span></
<input name="isShowPot9"><span>Up</span></
<input class="LinkedSpeed" value="1000 Full">
<input class="LinkedSpeed" value="Down">
`
	ports := parsePortStatus(body, 2)
	if len(ports) != 2 {
		t.Fatalf("len = %d", len(ports))
	}
	if !ports[0].Up || ports[0].SpeedMbps != 1000 {
		t.Fatalf("port1 = %+v", ports[0])
	}
	if ports[1].Up || ports[1].SpeedMbps != 0 {
		t.Fatalf("port2 = %+v", ports[1])
	}
	if ports[0].PoEAdmin {
		t.Fatal("PoEAdmin default should be false until PoE config is scraped")
	}
}

func TestGS3xxUint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		hi, lo string
		want   uint64
	}{
		{"0", "42", 42},
		{"1", "0", 1 << 32},
		{"1", "5", (1 << 32) | 5},
		{" ", "7", 7},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.hi+"_"+tt.lo, func(t *testing.T) {
			t.Parallel()
			if got := gs3xxUint(tt.hi, tt.lo); got != tt.want {
				t.Fatalf("gs3xxUint(%q,%q) = %d, want %d", tt.hi, tt.lo, got, tt.want)
			}
		})
	}
}

func TestParsePortStatistics(t *testing.T) {
	t.Parallel()

	ports := []Port{{Index: 1}, {Index: 2}}
	parsePortStatistics(`<input value="0">`, ports)
	if ports[0].RxBytes != 0 {
		t.Fatal("short input should leave ports unchanged")
	}

	vals := make([]string, 0, 12)
	// port1: rx hi/lo, tx hi/lo, crc hi/lo
	for _, n := range []string{"0", "10", "0", "20", "0", "1", "1", "0", "0", "5", "0", "2"} {
		vals = append(vals, `<input value="`+n+`">`)
	}
	body := strings.Join(vals, "")
	parsePortStatistics(body, ports)
	if ports[0].RxBytes != 10 || ports[0].TxBytes != 20 || ports[0].CRCErrors != 1 {
		t.Fatalf("port1 stats = %+v", ports[0])
	}
	if ports[1].RxBytes != 1<<32 || ports[1].TxBytes != 5 || ports[1].CRCErrors != 2 {
		t.Fatalf("port2 stats = %+v", ports[1])
	}

	ports = []Port{{Index: 1}}
	need := make([]string, 6)
	for i, n := range []string{"0", "3", "0", "4", "0", "5"} {
		need[i] = `<input value="` + n + `">`
	}
	wrapped := `noise<input value="9">` + `<div id="settingsStatusContainer">` + strings.Join(need, "") + `</div>`
	parsePortStatistics(wrapped, ports)
	if ports[0].RxBytes != 3 || ports[0].TxBytes != 4 || ports[0].CRCErrors != 5 {
		t.Fatalf("container stats = %+v", ports[0])
	}
}

func TestParsePoEConfigAndStatus(t *testing.T) {
	t.Parallel()

	ports := []Port{{Index: 1}, {Index: 2}, {Index: 3}}
	cfgBody := `
<li class="poe_port_list_item">
  <input class="port" value="1">
  <input class="hidPortPwr" value="1">
</li>
<li class="poe_port_list_item">
  <input class="port" value="2">
  <input class="hidPortPwr" value="0">
</li>
<li class="poe_port_list_item">
  <input class="port" value="99">
  <input class="hidPortPwr" value="1">
</li>
`
	parsePoEConfig(cfgBody, ports)
	if !ports[0].PoEAdmin || ports[1].PoEAdmin {
		t.Fatalf("admin = %v/%v", ports[0].PoEAdmin, ports[1].PoEAdmin)
	}

	statusBody := `
<li class="poe_port_list_item">
  <input class="port" value="1">
  <div class="poe-power-mode"><span> Delivering Power </span></div>
  <div class="powClassShow">@4@</div>
  <div class="hid_info_title"><span>ml570</span></div><div><span>48</span>
  <div class="hid_info_title"><span>ml572</span></div><div><span>120</span>
  <div class="hid_info_title"><span>ml574</span></div><div><span>5.5</span>
</li>
<li class="poe_port_list_item">
  <input class="port" value="2">
  <div class="poe-power-mode"><span>Searching</span></div>
  <div class="powClassShow">none</div>
</li>
`
	parsePoEStatus(statusBody, ports)
	if ports[0].PoEStatus != "delivering_power" || ports[0].PoEClass != 4 {
		t.Fatalf("port1 status = %+v", ports[0])
	}
	if ports[0].PoEVoltage != 48 || ports[0].PoECurrent != 120 || ports[0].PoEPowerW != 5.5 {
		t.Fatalf("port1 power = %+v", ports[0])
	}
	if ports[1].PoEStatus != "searching" || ports[1].PoEClass != 0 {
		t.Fatalf("port2 status = %+v", ports[1])
	}
}
