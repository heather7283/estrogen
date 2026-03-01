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

func handleDir(ctx context.Context, srcPath, dstPath string, opsChan chan<- Operation) {
	srcPathAbs := fp.Join(cfg.Src, srcPath)
	dstPathAbs := fp.Join(cfg.Dst, dstPath)

	entries, err := os.ReadDir(srcPathAbs)
	if err != nil {
		log.Printf("ERROR: failed to ReadDir %s: %v", srcPathAbs, err)
		return
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
				fmt.Printf("ERROR: empty dir name after rename (was %s)", srcName)
				return
			} else if strings.ContainsRune(dstName, '/') {
				fmt.Printf("ERROR: invalid dir name after rename: %s (was %s)", dstName, srcName)
				return
			} else if srcName2, exists := dstNames[dstName]; exists {
				log.Printf("ERROR: duplicate dir name after rename: %s (1st was %s, 2nd was %s)",
					dstName, srcName, srcName2)
				return
			} else {
				append2(&dirs, dirInfo{fp.Join(srcPath, srcName), fp.Join(dstPath, dstName)})
				dstNames[dstName] = srcName
				continue
			}
		}

		dstName, command, hasRule := findRule(dstName)
		if len(dstName) == 0 {
			fmt.Printf("ERROR: empty file name after applying rule (was %s)", srcName)
			return
		} else if strings.ContainsRune(dstName, '/') {
			fmt.Printf("ERROR: invalid file name after applying rule: %s (was %s)",
				dstName, srcName)
			return
		} else if srcName2, exists := dstNames[dstName]; exists {
			log.Printf("ERROR: duplicate file name after applying rule: %s (1st was %s, 2nd %s)",
				dstName, srcName, srcName2)
			return
		} else {
			dstNames[dstName] = srcName
		}

		src := fp.Join(srcPathAbs, srcName)
		dst := fp.Join(dstPathAbs, dstName)
		if hasRule {
			if isOlder, err := isOlderThan(dst, src); err != nil {
				log.Printf("ERROR: %v", err)
				continue
			} else if isOlder {
				append2(&ops, makeConvertOp(src, dst, command))
			} else {
				continue
			}
		} else if cfg.Settings.CopyUnmatched {
			if isOlder, err := isOlderThan(dst, src); err != nil {
				log.Printf("ERROR: %v", err)
				continue
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
		if dstEntries, err := os.ReadDir(dstPathAbs); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("ERROR: failed to ReadDir %s: %v", dstPathAbs, err)
			}
		} else {
			for _, dstEntry := range dstEntries {
				if _, exists := dstNames[dstEntry.Name()]; !exists {
					append2(&ops, makeDeleteOp(fp.Join(dstPathAbs, dstEntry.Name())))
				}
			}
		}
	}

	for _, op := range ops {
		opsChan <- op
	}

	for _, dir := range dirs {
		handleDir(ctx, dir.srcPath, dir.dstPath, opsChan)
	}
}

func Walker(ctx context.Context, origin string, opsChan chan<- Operation) {
	defer close(opsChan)

	handleDir(ctx, "", "", opsChan)
}

