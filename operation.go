package main

import (
	"context"
)

type Operation interface {
	Perform(ctx context.Context, dryRun bool) error
}

