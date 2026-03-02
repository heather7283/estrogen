package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	sc "strconv"
	"strings"
	"sync"
	"syscall"
)

var cfg *Config

func main() {
	var err error

	log.Default().SetFlags(log.Lshortfile)

	// command line arguments
	var (
		validateConfig bool
		configPath string
		nJobs uint
	)

	flag.BoolVar(&validateConfig, "validate", false, "Validate config and exit")
	flag.StringVar(&configPath, "config", "./estrogen.toml", "Path to config .toml")
	flag.UintVar(&nJobs, "j", uint(runtime.NumCPU()), "Number of parallel jobs to run")
	flag.Parse()

	if nJobs < 1 {
		log.Fatalf("Number of j*bs must be >= 1")
	}

	ctx, ctxCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if cfg, err = ParseConfig(configPath); err != nil {
		log.Fatalf("Error parsing config: %v", err)
	} else {
		log.Printf("Loaded config from %s", configPath)
		log.Printf("Src dir: %s", cfg.Src)
		log.Printf("Dst dir: %s", cfg.Dst)

		log.Printf("Settings:")
		log.Printf("      delete_removed: %v", cfg.Settings.DeleteRemoved)
		log.Printf("      copy_unmatched: %v", cfg.Settings.CopyUnmatched)
		log.Printf("      exclude_by_default: %v", cfg.Settings.ExcludeByDefault)
		log.Printf("      preserve_config_file: %v", cfg.Settings.PreserveConfigFile)

		log.Println("Filters:")
		for i, filter := range cfg.Filters {
			log.Printf("%4d: type: %v, pattern: %s",
				i + 1, filter.Type, filter.Regex.String())
		}

		log.Println("Renames:")
		for i, rename := range cfg.Renames {
			log.Printf("%4d: pattern: %s, replacement: %s",
				i + 1, rename.Pattern, rename.Replacement)
		}

		log.Println("Conversion rules:")
		for i, rule := range cfg.Rules {
			log.Printf("%4d: src: %s", i + 1, rule.Src.String())
			log.Printf("      dst: %s", rule.Dst)
			log.Printf("      cmd: %s", strings.Join(apply(rule.Cmd, sc.QuoteToGraphic), " "))
		}

		if (validateConfig) {
			os.Exit(0)
		}
	}

	opsChan := make(chan Operation, nJobs)

	wg := sync.WaitGroup{}
	wg.Go(func() { Walker(ctx, opsChan) })
	for range nJobs {
		wg.Go(func() { Worker(ctx, opsChan) })
	}

	wgDone := make(chan bool)
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-ctx.Done():
		log.Printf("Received termination signal")

		ctxCancel()
		log.Printf("Waiting for unfinished jobs...")
		wg.Wait()
	case <-wgDone:
		log.Printf("No more work to do, exiting")
	}
}

