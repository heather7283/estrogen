package main

import (
	"context"
)

type Operation interface {
	Perform(ctx context.Context) error
}

