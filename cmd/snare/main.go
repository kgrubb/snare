package main

import (
	"log"
	"os"
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

	store := agent.NewStore()
	traps := trap.New(cfg, store)

	go func() {
		for {
			snap, err := client.Poll()
			if err != nil {
				log.Printf("poll: %v", err)
			} else {
				traps.DiffAndSend(snap)
				store.Set(snap)
				log.Printf("scraped %s ports=%d", snap.Model, len(snap.Ports))
			}
			time.Sleep(cfg.Poll)
		}
	}()

	log.Printf("snare %s listening snmp on %s (community %q)", version, cfg.Listen, cfg.Community)
	if err := agent.ListenAndServe(cfg, store); err != nil {
		log.Fatal(err)
	}
}
