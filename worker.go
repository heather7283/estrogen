package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

func findRule(name string) (newName string, cmd []string, found bool) {
	for _, rule := range cfg.Rules {
		for _, submatches := range rule.Src.FindAllStringSubmatchIndex(name, -1) {
			var newName []byte
			newName = rule.Src.ExpandString(newName, rule.Dst, name, submatches)
			return string(newName), rule.Cmd, true
		}
	}

	return "", nil, false
}

func ProcessDir(ctx context.Context, dir Dir) {
	dstPathAbs := filepath.Join(cfg.Dst, dir.dstPath)
	srcPathAbs := filepath.Join(cfg.Src, dir.srcPath)

	if err := os.MkdirAll(dstPathAbs, 0o700); err != nil {
		log.Printf("ERROR: Could not mkdir %s: %v", dstPathAbs, err)
		return
	}

	for _, entry := range dir.entries {
		if entry.isDir {
			continue
		}

		var (
			dstName string
			command []string
		)
		if _dstName, _command, hasRule := findRule(entry.dstName); hasRule {
			dstName = _dstName
			command = _command
		} else if cfg.Settings.CopyUnmatched {
			dstName = entry.dstName
			// TODO: handle this in go
			command = []string{"cp", "@SRC@", "@DST@"}
		} else {
			continue
		}

		srcNameAbs := filepath.Join(srcPathAbs, entry.srcName)
		dstNameAbs := filepath.Join(dstPathAbs, dstName)

		srcStat, err := os.Stat(srcNameAbs)
		if err != nil {
			log.Printf("ERROR: Could not stat %s: %v", srcNameAbs, err)
			return
		}

		dstExists := true
		dstStat, err := os.Stat(dstNameAbs)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("ERROR: Could not stat %s: %v", dstNameAbs, err)
				return
			}
			dstExists = false
		}

		if !dstExists || srcStat.ModTime().After(dstStat.ModTime()) {
			log.Printf("CONV: %s -> %s", entry.srcName, dstName)
			command := makeCommand(command, srcNameAbs, dstNameAbs)
			if err := command.Run(); err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	}
}

func Worker(ctx context.Context, dirsChan <-chan Dir) {
	for dir := range dirsChan {
		ProcessDir(ctx, dir)
	}
}

