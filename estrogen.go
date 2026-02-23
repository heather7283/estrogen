package main

import (
	"context"
	"log"
	"os/signal"
	"runtime"
	"syscall"
)

var cfg *Config

func main() {
	var err error

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if cfg, err = ParseConfig(); err != nil {
		log.Fatalf("Error parsing config: %v", err)
	} else {
		log.Printf("Got config: %#v", cfg)
	}

	paths := make(chan Path, runtime.NumCPU())

	go Walk(ctx, cfg.Src, paths)

	outer:
	for {
		select {
		case <-ctx.Done():
			log.Printf("Received termination signal, exiting")
			break outer
		case path, open := <-paths:
			if !open {
				break outer
			}

			switch path.isDir {
			case true:
				log.Printf("Got directory: %v", path)
			case false:
				log.Printf("Got file: %v", path)
			}
		}
	}
}

