package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

var cfg *Config

func main() {
	var err error

	// command line arguments
	var (
		validateConfig bool = false
		configPath string = "./estrogen.toml"
	)

	flag.BoolVar(&validateConfig, "validate", false, "Validate config and exit")
	flag.StringVar(&configPath, "config", "./estrogen.toml", "Path to config .toml")
	flag.Parse()

	log.Default().SetFlags(log.Lshortfile)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if cfg, err = ParseConfig(configPath); err != nil {
		log.Fatalf("Error parsing config: %v", err)
	} else {
		log.Printf("Loaded config from %s", configPath)
		log.Printf("Src dir: %s", cfg.Src)
		log.Printf("Dst dir: %s", cfg.Dst)
		log.Printf("Loaded %d filters, %d renames, %d rules",
			len(cfg.Filters), len(cfg.Renames), len(cfg.Rules))
		log.Printf("Settings: delete_removed=%v copy_unmatched=%v exclude_by_default=%v",
			cfg.Settings.DeleteRemoved, cfg.Settings.CopyUnmatched, cfg.Settings.ExcludeByDefault)

		if (validateConfig) {
			os.Exit(0)
		}
	}

	numWorkers := runtime.NumCPU()

	paths := make(chan Path, numWorkers)

	go Walk(ctx, cfg.Src, paths)

	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	for range numWorkers {
		wg.Go(func() {
			Worker(ctx, paths)
		})
	}
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	outer:
	for {
		select {
		case <-ctx.Done():
			log.Printf("Received termination signal, exiting")
			break outer
		case <-wgDone:
			log.Printf("WaitGroup done, exiting")
			break outer
		}
	}
}

