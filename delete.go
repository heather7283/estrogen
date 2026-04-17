package main

import (
	"context"
	"log"
	"os"
	fp "path/filepath"
)

type deleteOperation struct {
	path string
}

func (o deleteOperation) Perform(ctx context.Context, dryRun bool) error {
	log.Printf("NUKE %s", fp.Base(o.path))
	if dryRun {
		return nil
	}

	return os.RemoveAll(o.path)
}

func makeDeleteOp(path string) Operation {
	return deleteOperation{
		path: path,
	}
}

