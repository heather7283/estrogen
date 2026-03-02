package main

import (
	"context"
	"fmt"
	"log"
	"os"
	fp "path/filepath"
	"strings"
)

func isFiltered(entry os.DirEntry) bool {
	for _, filter := range cfg.Filters {
		name := entry.Name()
		if (entry.IsDir()) {
			name += "/"
		}

		if !filter.Regex.MatchString(name) {
			continue
		}

		switch filter.Type {
		case FilterTypeInclude:
			return false
		case FilterTypeExclude:
			return true
		}
	}

	return cfg.Settings.ExcludeByDefault
}

func applyRenames(name string) string {
	for _, rename := range cfg.Renames {
		name = rename.Pattern.ReplaceAllString(name, rename.Replacement)
	}
	return name
}

func findRule(name string) (newName string, cmd []string, found bool) {
	for _, rule := range cfg.Rules {
		for _, submatches := range rule.Src.FindAllStringSubmatchIndex(name, -1) {
			var newName []byte
			newName = rule.Src.ExpandString(newName, rule.Dst, name, submatches)
			return string(newName), rule.Cmd, true
		}
	}

	return name, nil, false
}

func isOlderThan(path, reference string) (bool, error) {
	refStat, err := os.Stat(reference)
	if err != nil {
		return false, fmt.Errorf("Could not stat %s: %v", reference, err)
	}

	pathExists := true
	pathStat, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("ERROR: Could not stat %s: %v", path, err)
		}
		pathExists = false
	}

	if !pathExists || refStat.ModTime().After(pathStat.ModTime()) {
		return true, nil
	} else {
		return false, nil
	}
}

func handleDir(ctx context.Context, srcPath, dstPath string, opsChan chan<- Operation) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("failed to ReadDir %s: %v", srcPath, err)
	}

	type dirInfo struct { srcPath, dstPath string }
	dirs := make([]dirInfo, 0)

	ops := make([]Operation, 0)

	// for finding duplicates, srcName = dstNames[dstName]
	dstNames := make(map[string]string)
	for _, entry := range entries {
		if isFiltered(entry) {
			continue
		}

		srcName := entry.Name()
		dstName := applyRenames(srcName)

		if entry.IsDir() {
			if len(dstName) == 0 {
				return fmt.Errorf("empty dir name after rename (was %s)", srcName)
			} else if strings.ContainsRune(dstName, '/') {
				return fmt.Errorf("invalid dir name after rename: %s (was %s)", dstName, srcName)
			} else if srcName2, exists := dstNames[dstName]; exists {
				return fmt.Errorf("duplicate dir name after rename: %s (1st was %s, 2nd was %s)",
					dstName, srcName, srcName2)
			} else {
				append2(&dirs, dirInfo{fp.Join(srcPath, srcName), fp.Join(dstPath, dstName)})
				dstNames[dstName] = srcName
				continue
			}
		}

		dstName, command, hasRule := findRule(dstName)
		if len(dstName) == 0 {
			return fmt.Errorf("empty file name after applying rule (was %s)", srcName)
		} else if strings.ContainsRune(dstName, '/') {
			return fmt.Errorf("invalid file name after applying rule: %s (was %s)",
				dstName, srcName)
		} else if srcName2, exists := dstNames[dstName]; exists {
			return fmt.Errorf("duplicate file name after applying rule: %s (1st was %s, 2nd %s)",
				dstName, srcName, srcName2)
		} else {
			dstNames[dstName] = srcName
		}

		src := fp.Join(srcPath, srcName)
		dst := fp.Join(dstPath, dstName)
		if hasRule {
			if isOlder, err := isOlderThan(dst, src); err != nil {
				return err
			} else if isOlder {
				append2(&ops, makeConvertOp(src, dst, command))
			} else {
				continue
			}
		} else if cfg.Settings.CopyUnmatched {
			if isOlder, err := isOlderThan(dst, src); err != nil {
				return err
			} else if isOlder {
				append2(&ops, makeCopyOp(src, dst))
			} else {
				continue
			}
		} else {
			continue
		}
	}

	if cfg.Settings.DeleteRemoved {
		if dstEntries, err := os.ReadDir(dstPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to ReadDir %s: %v", dstPath, err)
			}
		} else {
			for _, dstEntry := range dstEntries {
				name := dstEntry.Name()
				if cfg.Settings.PreserveConfigFile && name == cfg.ConfigFileName {
					continue
				} else if _, exists := dstNames[name]; !exists {
					append2(&ops, makeDeleteOp(fp.Join(dstPath, name)))
				}
			}
		}
	}

	for _, op := range ops {
		opsChan <- op
	}

	for _, dir := range dirs {
		if err := handleDir(ctx, dir.srcPath, dir.dstPath, opsChan); err != nil {
			return err
		}
	}

	return nil
}

func Walker(ctx context.Context, opsChan chan<- Operation) {
	defer close(opsChan)

	if err := handleDir(ctx, cfg.Src, cfg.Dst, opsChan); err != nil {
		log.Printf("ERROR: %v", err)
	}
}

