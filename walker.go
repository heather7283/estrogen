package main

import (
	"context"
	"log"
	"os"
	fp "path/filepath"
	"slices"
	"strings"
)

type Dir struct {
	srcPath, dstPath string // relative
	entries []DirEntry
}

type DirEntry struct {
	srcName, dstName string
	isDir bool
}

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

// origin - source path
// prefix - path relative to origin
// name - name of current directory we're handling
func HandleDir(ctx context.Context, srcPath, dstPath string, dirsChan chan<- Dir) {
	srcPathAbs := fp.Join(cfg.Src, srcPath)

	entries, err := os.ReadDir(srcPathAbs)
	if err != nil {
		log.Printf("ERROR: failed to ReadDir %s: %v", srcPathAbs, err)
		return
	}

	renamedDirs := make([]DirEntry, 0, len(entries))
	renamedFiles := make([]DirEntry, 0, len(entries))

	// indexed by new name, using a map to catch duplicate names
	dstNames := make(map[string]string)
	for _, entry := range entries {
		if isFiltered(entry) {
			continue
		}

		srcName := entry.Name()
		dstName := applyRenames(srcName)
		if len(dstName) == 0 {
			log.Printf("ERROR: empty name after rename, was %s", srcName)
			return
		} else if strings.ContainsRune(dstName, '/') {
			log.Printf("ERROR: invalid name after rename: %s (was %s)", dstName, srcName)
			return
		} else if srcName2, exists := dstNames[dstName]; exists {
			log.Printf("ERROR: duplicate name after rename: %s (1st was %s, 2nd was %s)",
				dstName, srcName, srcName2)
			return
		} else {
			dstNames[dstName] = srcName
			if entry.IsDir() {
				append2(&renamedDirs, DirEntry{srcName: srcName, dstName: dstName, isDir: true})
			} else {
				append2(&renamedFiles, DirEntry{srcName: srcName, dstName: dstName, isDir: false})
			}
		}
	}

	for _, dir := range renamedDirs {
		HandleDir(ctx, fp.Join(srcPath, dir.srcName), fp.Join(dstPath, dir.dstName), dirsChan)
	}

	dirsChan <- Dir{
		srcPath: srcPath,
		dstPath: dstPath,
		entries: slices.Concat(renamedDirs, renamedFiles),
	}
}

func Walker(ctx context.Context, origin string, dirChan chan<- Dir) {
	defer close(dirChan)

	HandleDir(ctx, "", "", dirChan)
}

