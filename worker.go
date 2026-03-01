package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type opType int

const (
	opTypeCopy opType = iota
	opTypeDelete opType = iota
	opTypeConvert opType = iota
)

type Operation struct {
	opType opType
	srcPath, dstPath string
	command []string
}

func makeCommand(cmd []string, src, dst string) *exec.Cmd {
	argv := make([]string, len(cmd))
	for i := range cmd {
		r := strings.NewReplacer("@SRC@", src, "@DST@", dst)
		argv[i] = r.Replace(cmd[i])
	}

	command := exec.Command(argv[0], argv[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command
}

func handleOp(ctx context.Context, op Operation) {
	switch op.opType {
	case opTypeConvert:
		log.Printf("CONV: %s -> %s",
			strings.TrimPrefix(op.srcPath, cfg.Src), strings.TrimPrefix(op.dstPath, cfg.Dst))

		dir := filepath.Dir(op.dstPath)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			log.Printf("ERROR: failed to create directory %s: %v", dir, err)
			return
		}

		command := makeCommand(op.command, op.srcPath, op.dstPath)
		if err := command.Run(); err != nil {
			log.Printf("ERROR: %v", err)
			return
		}
	case opTypeCopy:
		log.Printf("COPY: %s -> %s",
			strings.TrimPrefix(op.srcPath, cfg.Src), strings.TrimPrefix(op.dstPath, cfg.Dst))

		dir := filepath.Dir(op.dstPath)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			log.Printf("ERROR: failed to create directory %s: %v", dir, err)
			return
		}

		srcFile, err := os.Open(op.srcPath)
		if err != nil {
			log.Printf("ERROR: failed to open %s: %v", op.srcPath, err)
			return
		}
		defer srcFile.Close()

		dstFile, err := os.Create(op.dstPath)
		if err != nil {
			log.Printf("ERROR: failed to create %s: %v", op.dstPath, err)
			return
		}
		defer dstFile.Close()

		if _, err = io.Copy(dstFile, srcFile); err != nil {
			log.Printf("ERROR: failed to copy from %s to %s: %v", op.srcPath, op.dstPath, err)
			if err := os.Remove(op.dstPath); err != nil {
				log.Printf("ERROR: failed to delete %s: %v", op.dstPath, err)
			}
			return
		}
	case opTypeDelete:
		log.Printf(" DEL: %s", strings.TrimPrefix(op.dstPath, cfg.Dst))

		if err := os.RemoveAll(op.dstPath); err != nil {
			log.Printf("ERROR: failed to delete %s: %v", op.dstPath, err)
			return
		}
	}
}

func Worker(ctx context.Context, opsChan <-chan Operation) {
	for dir := range opsChan {
		handleOp(ctx, dir)
	}
}

