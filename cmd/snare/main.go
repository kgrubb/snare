package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kgrubb/snare/internal/agent"
	"github.com/kgrubb/snare/internal/config"
	"github.com/kgrubb/snare/internal/netgear"
	"github.com/kgrubb/snare/internal/trap"
	"github.com/spf13/pflag"
)

var version = "dev"

func main() {
	cfgPath := pflag.String("config", "config.yaml", "path to config file")
	showVer := pflag.Bool("version", false, "print version")
	pflag.Parse()
	if *showVer {
		log.Printf("snare %s", version)
		os.Exit(0)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	client, err := netgear.NewClient(cfg.Switch.URL, cfg.Switch.Password, cfg.Switch.Ports, cfg.Switch.PoEBudgetWatts)
	if err != nil {
		log.Fatalf("switch client: %v", err)
	}

	// Free the switch session on SIGTERM so k8s rollouts do not orphan the only SID slot.
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
		<-ch
		client.Logout()
		os.Exit(0)
	}()

	store := agent.NewStore()
	traps := trap.New(cfg, store)

	go func() {
		wait := cfg.Poll
		for {
			snap, err := client.Poll()
			if err != nil {
				log.Printf("poll: %v", err)
				// Only slow down for a full session table so it can idle-expire.
				if strings.Contains(err.Error(), "session limit") {
					wait *= 2
					if wait > 10*time.Minute {
						wait = 10 * time.Minute
					}
				} else {
					wait = cfg.Poll
				}
			} else {
				wait = cfg.Poll
				traps.DiffAndSend(snap)
				store.Set(snap)
				log.Printf("scraped %s ports=%d", snap.Model, len(snap.Ports))
			}
			time.Sleep(wait)
		}
	}()

	log.Printf("snare %s listening snmp on %s (community %q)", version, cfg.Listen, cfg.Community)
	if err := agent.ListenAndServe(cfg, store); err != nil {
		log.Fatal(err)
	}
}
