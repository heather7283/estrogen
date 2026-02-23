package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
)

type Path struct {
	prefix, name string
	isDir bool
}

func IsExcluded(entry os.DirEntry) bool {
	for _, filter := range cfg.Filters {
		name := entry.Name()
		if (entry.IsDir()) {
			name += "/"
		}

		if !filter.Re.MatchString(name) {
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

func HandleDir(ctx context.Context, prefix, name string, paths chan<- Path) {
	dir := filepath.Join(prefix, name)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("HandleDir: failed to ReadDir: %v", err)
		return
	}

	for _, entry := range entries {
		if IsExcluded(entry) {
			continue
		}

		paths <-Path{dir, entry.Name(), entry.IsDir()}

		if entry.IsDir() {
			HandleDir(ctx, dir, entry.Name(), paths)
		}
	}
}

func Walk(ctx context.Context, origin string, paths chan<- Path) {
	defer close(paths)

	HandleDir(ctx, origin, ".", paths)
}

