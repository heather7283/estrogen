package main

import (
	"context"
	"log"
	"os"
	"io"
	fp "path/filepath"
)

type copyOperation struct {
	src, dst string
}

func (o copyOperation) Perform(ctx context.Context) error {
	log.Printf("COPY %s -> %s", fp.Base(o.src), fp.Base(o.dst))

	dir := fp.Dir(o.dst)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	srcFile, err := os.Open(o.src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(o.dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

func makeCopyOp(src, dst string) Operation {
	return copyOperation{
		src: src,
		dst: dst,
	}
}

