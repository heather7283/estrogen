package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunCommand(cmd Command, src, dst string) (*os.Process, error) {
	newCmd := make([]string, len(cmd))
	for i := range cmd {
		r := strings.NewReplacer("@SRC@", src, "@DST@", dst)
		newCmd[i] = r.Replace(cmd[i])
	}

	if executable, err := exec.LookPath(newCmd[0]); err != nil {
		return nil, err
	} else {
		return os.StartProcess(executable, newCmd, &os.ProcAttr{})
	}
}

func MatchRule(name string) (string, Command, bool) {
	for _, rule := range cfg.Rules {
		for _, submatches := range rule.SrcRe.FindAllStringSubmatchIndex(name, -1) {
			var newName []byte
			newName = rule.SrcRe.ExpandString(newName, rule.Dst, name, submatches)
			return string(newName), rule.Cmd, true
		}
	}

	return "", nil, false
}

func ProcessPath(ctx context.Context, path Path) {
	srcBase := cfg.Src
	dstBase := cfg.Dst

	if path.isDir {
		dir := filepath.Join(dstBase, path.prefix, path.name)
		err := os.Mkdir(dir, 0o700)
		if err != nil {
			if !os.IsExist(err) {
				log.Printf("ERROR: %v", err)
			}
		} else {
			log.Printf("Worker: created directory: %s", dir)
		}
		return
	}

	srcName := path.name
	dstName, cmd, matched := MatchRule(srcName)
	if !matched {
		if !cfg.Settings.CopyUnmatched {
			return
		}
		dstName = path.name
		// TODO: handle this in go
		cmd = Command{"cp", "@SRC@", "@DST@"}
	}

	//srcExists := true
	srcPath := filepath.Join(srcBase, path.prefix, path.name)
	srcStat, err := os.Stat(srcPath)
	if err != nil {
		//if !os.IsNotExist(err) {
			log.Printf("ERROR: failed to stat %s: %v", srcPath, err)
			return
		//}
		//srcExists = false
	}

	dstExists := true
	dstPath := filepath.Join(dstBase, path.prefix, dstName)
	dstStat, err := os.Stat(dstPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("ERROR: failed to stat %s: %v", dstPath, err)
			return
		}
		dstExists = false
	}

	if !dstExists || srcStat.ModTime().After(dstStat.ModTime()) {
		log.Printf("Worker: %s -> %s (%s)", srcName, dstName, cmd[0])
		proc, err := RunCommand(cmd, srcPath, dstPath)
		if err != nil {
			log.Printf("ERROR: run %s: %v", cmd[0], err)
		} else if _, err := proc.Wait(); err != nil {
			log.Printf("ERROR: command returned: %v", err)
		}
	}
}

func Worker(ctx context.Context, paths <-chan Path) {
	for path := range paths {
		ProcessPath(ctx, path)
	}
}

