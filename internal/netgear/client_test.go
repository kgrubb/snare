package netgear

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		ports   int
		budget  uint
		wantErr bool
		portsN  int
		budgetN uint
	}{
		{name: "defaults", url: "http://switch.local/", ports: 0, budget: 0, portsN: 8, budgetN: 62},
		{name: "explicit", url: "http://192.168.1.1", ports: 16, budget: 120, portsN: 16, budgetN: 120},
		{name: "bad url", url: "://bad", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := NewClient(tt.url, "pw", tt.ports, tt.budget)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.ports != tt.portsN || c.budget != tt.budgetN {
				t.Fatalf("ports/budget = %d/%d, want %d/%d", c.ports, c.budget, tt.portsN, tt.budgetN)
			}
			if strings.HasSuffix(c.base, "/") {
				t.Fatalf("base should trim trailing slash: %q", c.base)
			}
		})
	}
}

func TestClientPollAndLogin(t *testing.T) {
	t.Parallel()

	const dash = `
<html><head><title>GS308EP</title></head>
<body>
<span id="switch_name">lab</span>
<span>ml198</span></div><div><span>SER1</span>
<span>ml089</span></div><div><span>1.0</span>
<input name="isShowPot1"><span>Up</span></
<input class="LinkedSpeed" value="1000 Full">
</body></html>`

	stats := `<div id="settingsStatusContainer">` +
		`<input value="0"><input value="11"><input value="0"><input value="22"><input value="0"><input value="0">` +
		`</div>`

	poeCfg := `
<li class="poe_port_list_item">
  <input class="port" value="1">
  <input class="hidPortPwr" value="1">
</li>`

	poeStatus := `
<li class="poe_port_list_item">
  <input class="port" value="1">
  <div class="poe-power-mode"><span>Delivering Power</span></div>
  <div class="powClassShow">@3@</div>
  <div class="hid_info_title"><span>ml570</span></div><div><span>54</span>
  <div class="hid_info_title"><span>ml572</span></div><div><span>100</span>
  <div class="hid_info_title"><span>ml574</span></div><div><span>4.2</span>
</li>`

	logins := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/login.cgi", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`<input id="rand" value="12">`))
			return
		}
		logins++
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "ok"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/dashboard.cgi", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(dash))
	})
	mux.HandleFunc("/portStatistics.cgi", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(stats))
	})
	mux.HandleFunc("/PoEPortConfig.cgi", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(poeCfg))
	})
	mux.HandleFunc("/getPoePortStatus.cgi", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(poeStatus))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, "ab", 1, 62)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := c.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if snap.Model != "GS308EP" || snap.Name != "lab" || snap.Serial != "SER1" || snap.Firmware != "1.0" {
		t.Fatalf("snap meta = %+v", snap)
	}
	if len(snap.Ports) != 1 || !snap.Ports[0].Up || snap.Ports[0].RxBytes != 11 || snap.Ports[0].PoEClass != 3 {
		t.Fatalf("ports = %+v", snap.Ports)
	}
	if snap.PoEBudget != 62 {
		t.Fatalf("budget = %d", snap.PoEBudget)
	}
	if _, err := c.Poll(); err != nil {
		t.Fatal(err)
	}
	if logins != 1 {
		t.Fatalf("logins = %d, want 1 (reuse SID)", logins)
	}
}

func TestClientLoginFailures(t *testing.T) {
	t.Parallel()

	t.Run("no sid cookie", func(t *testing.T) {
		t.Parallel()
		mux := http.NewServeMux()
		mux.HandleFunc("/login.cgi", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`<html></html>`))
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		c, err := NewClient(srv.URL, "pw", 8, 62)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.login(); err == nil {
			t.Fatal("expected login failure")
		}
	})

	t.Run("relogin when sid expired", func(t *testing.T) {
		t.Parallel()
		logins := 0
		authed := false
		mux := http.NewServeMux()
		mux.HandleFunc("/login.cgi", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`<input id='rand' value='1'>`))
				return
			}
			logins++
			authed = true
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "ok"})
		})
		mux.HandleFunc("/dashboard.cgi", func(w http.ResponseWriter, r *http.Request) {
			if !authed {
				_, _ = w.Write([]byte(`<input id='rand' value='9'>`))
				return
			}
			_, _ = w.Write([]byte(`<html><head><title>GS308EP</title></head><body><span id="switch_name">lab</span></body></html>`))
		})
		mux.HandleFunc("/portStatistics.cgi", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<html></html>`))
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		c, err := NewClient(srv.URL, "pw", 1, 62)
		if err != nil {
			t.Fatal(err)
		}
		base, _ := url.Parse(srv.URL)
		c.http.Jar.SetCookies(base, []*http.Cookie{{Name: "SID", Value: "stale"}})
		authed = false
		if _, err := c.Poll(); err != nil {
			t.Fatal(err)
		}
		if logins != 1 {
			t.Fatalf("logins = %d, want 1", logins)
		}
	})

	t.Run("stale sid does not mask failed relogin", func(t *testing.T) {
		t.Parallel()
		mux := http.NewServeMux()
		mux.HandleFunc("/login.cgi", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<input id='rand' value='1'>`))
		})
		mux.HandleFunc("/dashboard.cgi", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<input id='rand' value='9'>`))
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		c, err := NewClient(srv.URL, "pw", 1, 62)
		if err != nil {
			t.Fatal(err)
		}
		base, _ := url.Parse(srv.URL)
		c.http.Jar.SetCookies(base, []*http.Cookie{{Name: "SID", Value: "stale"}})
		if _, err := c.Poll(); err == nil {
			t.Fatal("expected poll failure when relogin sets no SID")
		}
	})

	t.Run("http error on get", func(t *testing.T) {
		t.Parallel()
		mux := http.NewServeMux()
		mux.HandleFunc("/login.cgi", func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "ok"})
			_, _ = w.Write([]byte(`ok`))
		})
		mux.HandleFunc("/dashboard.cgi", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusInternalServerError)
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		c, err := NewClient(srv.URL, "pw", 8, 62)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Poll(); err == nil {
			t.Fatal("expected poll error")
		}
	})

	t.Run("default model when title missing", func(t *testing.T) {
		t.Parallel()
		mux := http.NewServeMux()
		mux.HandleFunc("/login.cgi", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				http.SetCookie(w, &http.Cookie{Name: "GS108SID", Value: "ok"})
			}
			_, _ = w.Write([]byte(`ok`))
		})
		mux.HandleFunc("/dashboard.cgi", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<html><body></body></html>`))
		})
		mux.HandleFunc("/portStatistics.cgi", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<html></html>`))
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		c, err := NewClient(srv.URL, "pw", 1, 62)
		if err != nil {
			t.Fatal(err)
		}
		snap, err := c.Poll()
		if err != nil {
			t.Fatal(err)
		}
		if snap.Model != "GS308EP" {
			t.Fatalf("model = %q", snap.Model)
		}
	})
}
