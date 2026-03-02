package main

import (
	"log"
	"context"
)

func Worker(ctx context.Context, opsChan <-chan Operation) {
	for {
		if ctx.Err() != nil {
			return
		}

		select {
		case op, ok := <-opsChan:
			if !ok {
				return
			} else if err := op.Perform(ctx); err != nil {
				log.Printf("ERROR: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

