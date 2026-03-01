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

	dstDir := fp.Dir(o.dst)
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return err
	}

	srcFile, err := os.Open(o.src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	tmpFile, err := tmpfile(dstDir, fp.Base(o.dst))
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err = io.Copy(tmpFile, srcFile); err != nil {
		return err
	}

	if err := os.Rename(tmpFile.Name(), o.dst); err != nil {
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

