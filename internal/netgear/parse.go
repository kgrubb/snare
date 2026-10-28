package netgear

import (
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var (
	reRand     = regexp.MustCompile(`(?i)id=["']rand["'][^>]*value=["']([^"']+)["']|value=["']([^"']+)["'][^>]*id=["']rand["']`)
	reHash     = regexp.MustCompile(`(?i)name=["']hash["'][^>]*value=["']([^"']+)["']|value=["']([^"']+)["'][^>]*name=["']hash["']`)
	reName     = regexp.MustCompile(`(?is)id=["']switch_name["'][^>]*>([^<]*)`)
	reSpeed    = regexp.MustCompile(`(?i)class=["']Speed["'][^>]*value=["']([^"']*)["']`)
	reLinked   = regexp.MustCompile(`(?i)class=["']LinkedSpeed["'][^>]*value=["']([^"']*)["']`)
	reShowPort = regexp.MustCompile(`(?is)name=["']isShowPot(\d+)["'][^>]*>[\s\S]*?<[^>]+>([^<]*)</`)
	reInputs   = regexp.MustCompile(`(?i)<input[^>]*value=["']([^"']*)["']`)
	rePoEItem  = regexp.MustCompile(`(?is)<li[^>]*class=["'][^"']*poe_port_list_item[^"']*["'][^>]*>(.*?)</li>`)
	rePoEPort  = regexp.MustCompile(`(?i)class=["']port["'][^>]*value=["'](\d+)["']`)
	rePoEPwr   = regexp.MustCompile(`(?i)class=["']hidPortPwr["'][^>]*value=["'](\d+)["']`)
	rePoEMode  = regexp.MustCompile(`(?is)class=["'][^"']*poe-power-mode[^"']*["'][^>]*>\s*<span[^>]*>\s*([^<]+)\s*<`)
	rePoEClass = regexp.MustCompile(`(?is)class=["'][^"']*powClassShow[^"']*["'][^>]*>\s*([^<]*)`)
	reAtClass  = regexp.MustCompile(`@(\d+)@`)
	reMLLabel  = regexp.MustCompile(`(?is)class=["']hid_info_title["'][^>]*>\s*<span[^>]*>([^<]+)</span>\s*</div>\s*<div[^>]*>\s*<span[^>]*>([^<]*)</span>`)
)

func firstGroup(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	for i := 1; i < len(m); i++ {
		if m[i] != "" {
			return m[i]
		}
	}
	return ""
}

func parseRand(body string) string { return firstGroup(reRand, body) }
func parseHash(body string) string { return firstGroup(reHash, body) }

func parseTitleModel(body string) string {
	z := html.NewTokenizer(strings.NewReader(body))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		t := z.Token()
		if t.Type == html.StartTagToken && t.Data == "title" {
			if z.Next() == html.TextToken {
				return strings.TrimSpace(z.Token().Data)
			}
		}
	}
	return ""
}

func afterLabel(body, label string) string {
	idx := strings.Index(body, ">"+label+"<")
	if idx < 0 {
		return ""
	}
	re := regexp.MustCompile(`(?is)</span>\s*</div>\s*<div[^>]*>\s*<span[^>]*>([^<]*)</span>`)
	if m := re.FindStringSubmatch(body[idx:]); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func parseMetadata(body string) (name, serial, fw string) {
	return strings.TrimSpace(firstGroup(reName, body)), afterLabel(body, "ml198"), afterLabel(body, "ml089")
}

func parsePortStatus(body string, n int) []Port {
	ports := make([]Port, n)
	for i := range ports {
		ports[i].Index = i + 1
		ports[i].PoEAdmin = true
	}
	for _, m := range reShowPort.FindAllStringSubmatch(body, -1) {
		idx, _ := strconv.Atoi(m[1])
		if idx < 1 || idx > n {
			continue
		}
		st := strings.ToLower(strings.TrimSpace(m[2]))
		ports[idx-1].Up = st == "up" || strings.Contains(st, "up")
	}
	linked := reLinked.FindAllStringSubmatch(body, -1)
	for i := 0; i < n && i < len(linked); i++ {
		ports[i].SpeedMbps = parseSpeedMbps(linked[i][1])
		if ports[i].SpeedMbps > 0 {
			ports[i].Up = true
		}
	}
	_ = reSpeed
	return ports
}

func parseSpeedMbps(s string) uint64 {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("full", "", "half", "", "duplex", "").Replace(s)
	s = strings.TrimSpace(s)
	switch {
	case strings.Contains(s, "1000"), strings.Contains(s, "1g"):
		return 1000
	case strings.Contains(s, "100"):
		return 100
	case s == "" || s == "0" || strings.Contains(s, "down"):
		return 0
	case strings.Contains(s, "10"):
		return 10
	}
	v, _ := strconv.ParseUint(strings.TrimRight(s, "mMbBgGtTpPsS/ "), 10, 64)
	return v
}

func parsePortStatistics(body string, ports []Port) {
	vals := reInputs.FindAllStringSubmatch(body, -1)
	need := len(ports) * 6
	if len(vals) < need {
		return
	}
	// Prefer the statistics container block when present.
	if i := strings.Index(body, `id="settingsStatusContainer"`); i >= 0 {
		vals = reInputs.FindAllStringSubmatch(body[i:], -1)
		if len(vals) < need {
			return
		}
	}
	for i := range ports {
		base := i * 6
		ports[i].RxBytes = gs3xxUint(vals[base][1], vals[base+1][1])
		ports[i].TxBytes = gs3xxUint(vals[base+2][1], vals[base+3][1])
		ports[i].CRCErrors = gs3xxUint(vals[base+4][1], vals[base+5][1])
	}
}

func gs3xxUint(hi, lo string) uint64 {
	h, _ := strconv.ParseUint(strings.TrimSpace(hi), 10, 64)
	l, _ := strconv.ParseUint(strings.TrimSpace(lo), 10, 64)
	return h<<32 | l
}

func parsePoEConfig(body string, ports []Port) {
	for _, m := range rePoEItem.FindAllStringSubmatch(body, -1) {
		idx, _ := strconv.Atoi(firstGroup(rePoEPort, m[1]))
		if idx < 1 || idx > len(ports) {
			continue
		}
		ports[idx-1].PoEAdmin = firstGroup(rePoEPwr, m[1]) == "1"
	}
}

func parsePoEStatus(body string, ports []Port) {
	for _, m := range rePoEItem.FindAllStringSubmatch(body, -1) {
		item := m[1]
		idx, _ := strconv.Atoi(firstGroup(rePoEPort, item))
		if idx < 1 || idx > len(ports) {
			continue
		}
		st := strings.ToLower(strings.TrimSpace(firstGroup(rePoEMode, item)))
		ports[idx-1].PoEStatus = strings.ReplaceAll(st, " ", "_")
		if c := firstGroup(reAtClass, firstGroup(rePoEClass, item)); c != "" {
			ports[idx-1].PoEClass, _ = strconv.Atoi(c)
		}
		labels := map[string]string{}
		for _, lm := range reMLLabel.FindAllStringSubmatch(item, -1) {
			labels[strings.TrimSpace(lm[1])] = strings.TrimSpace(lm[2])
		}
		ports[idx-1].PoEVoltage, _ = strconv.Atoi(labels["ml570"])
		ports[idx-1].PoECurrent, _ = strconv.Atoi(labels["ml572"])
		ports[idx-1].PoEPowerW, _ = strconv.ParseFloat(labels["ml574"], 64)
	}
}
