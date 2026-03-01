package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	fp "path/filepath"
)

type convertOperation struct {
	src, dst string
	cmd []string
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

func (o convertOperation) Perform(ctx context.Context) error {
	log.Printf("CONV %s -> %s", fp.Base(o.src), fp.Base(o.dst))

	dir := fp.Dir(o.dst)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	cmd := makeCommand(o.cmd, o.src, o.dst)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func makeConvertOp(src, dst string, cmd []string) Operation {
	return convertOperation{
		src: src,
		dst: dst,
		cmd: cmd,
	}
}

