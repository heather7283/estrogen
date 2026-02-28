package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	//"runtime"
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

		log.Printf("Loaded %d filters:", len(cfg.Filters))
		for i, filter := range cfg.Filters {
			log.Printf("\t%d: type: %v, pattern: %s", i, filter.Type, filter.Regex.String())
		}

		log.Printf("Loaded %d renames:", len(cfg.Renames))
		for i, rename := range cfg.Renames {
			log.Printf("\t%d: pattern: %s, replacement: %s", i, rename.Pattern, rename.Replacement)
		}

		log.Printf("Loaded %d rules:", len(cfg.Rules))
		for i, rule := range cfg.Rules {
			log.Printf("\t%d: src: %s, dst: %s, cmd: %v", i, rule.Src.String(), rule.Dst, rule.Cmd)
		}

		log.Printf("Settings:")
		log.Printf("\tdelete_removed: %v", cfg.Settings.DeleteRemoved)
		log.Printf("\tcopy_unmatched: %v", cfg.Settings.CopyUnmatched)
		log.Printf("\texclude_by_default: %v", cfg.Settings.ExcludeByDefault)

		if (validateConfig) {
			os.Exit(0)
		}
	}

	//numWorkers := runtime.NumCPU()
	numWorkers := 1

	dirsChan := make(chan Dir, numWorkers)

	go Walker(ctx, cfg.Src, dirsChan)

	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	for range numWorkers {
		wg.Go(func() {
			Worker(ctx, dirsChan)
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

