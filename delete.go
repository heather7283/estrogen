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

func (o deleteOperation) Perform(ctx context.Context) error {
	log.Printf("NUKE %s", fp.Base(o.path))
	return os.RemoveAll(o.path)
}

func makeDeleteOp(path string) Operation {
	return deleteOperation{
		path: path,
	}
}

