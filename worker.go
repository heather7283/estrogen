package main

import (
	"log"
	"context"
)

func Worker(ctx context.Context, opsChan <-chan Operation) {
	for op := range opsChan {
		if err := op.Perform(ctx); err != nil {
			log.Printf("ERROR: %v", err)
		}
	}
}

