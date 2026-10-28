package netgear

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client scrapes Netgear GS30xEP Plus web UI (same CGI surface as py-netgear-plus).
type Client struct {
	base     string
	password string
	ports    int
	budget   uint
	http     *http.Client
}

func NewClient(baseURL, password string, ports int, poeBudget uint) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	if ports <= 0 {
		ports = 8
	}
	if poeBudget == 0 {
		poeBudget = 62
	}
	return &Client{
		base:     u.String(),
		password: password,
		ports:    ports,
		budget:   poeBudget,
		http: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		},
	}, nil
}

func (c *Client) Poll() (*Snapshot, error) {
	if err := c.login(); err != nil {
		return nil, err
	}
	dash, err := c.get("/dashboard.cgi")
	if err != nil {
		return nil, err
	}
	name, serial, fw := parseMetadata(dash)
	ports := parsePortStatus(dash, c.ports)

	stats, err := c.get("/portStatistics.cgi")
	if err != nil {
		return nil, err
	}
	parsePortStatistics(stats, ports)

	if body, err := c.get("/PoEPortConfig.cgi"); err == nil {
		parsePoEConfig(body, ports)
	}
	if body, err := c.get("/getPoePortStatus.cgi"); err == nil {
		parsePoEStatus(body, ports)
	}

	model := parseTitleModel(dash)
	if model == "" {
		model = "GS308EP"
	}
	return &Snapshot{
		FetchedAt: time.Now().UTC(),
		Model:     model,
		Name:      name,
		Serial:    serial,
		Firmware:  fw,
		Ports:     ports,
		PoEBudget: c.budget,
	}, nil
}

func (c *Client) login() error {
	body, err := c.get("/login.cgi")
	if err != nil {
		return err
	}
	rand := parseRand(body)
	pass := c.password
	if rand != "" {
		pass = mergeHash(c.password, rand)
	}
	form := url.Values{"password": {pass}}
	req, err := http.NewRequest(http.MethodPost, c.base+"/login.cgi", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	u, _ := url.Parse(c.base)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "SID" || ck.Name == "GS108SID" {
			return nil
		}
	}
	return fmt.Errorf("login failed: no SID cookie (status %d)", resp.StatusCode)
}

func (c *Client) get(path string) (string, error) {
	resp, err := c.http.Get(c.base + path)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%s: HTTP %d", path, resp.StatusCode)
	}
	return string(b), nil
}
