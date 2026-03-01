package main

import (
	"fmt"
	"math/rand"
	"os"
	fp "path/filepath"
	"strings"
)

func append2[S ~[]E, E any](s *S, e ...E) {
	*s = append(*s, e...)
}

func tmpfile(dir, name string) (*os.File, error) {
	// can't use os.CreateTemp because it appends random string to the end
	// of file name, and things like ffmpeg and imagemagick rely on file
	// extension for filetype detection

	for range 100 {
		f, err := os.Create(fp.Join(dir, fmt.Sprintf("__tmp%06d.%s", rand.Intn(100500), name)))
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return nil, err
		}
		return f, nil
	}

	return nil, fmt.Errorf("Too many failed attempts")
}

func realpath(path string) (string, error) {
	if suffix, hasPrefix := strings.CutPrefix(path, "~/"); hasPrefix {
		if home, err := os.UserHomeDir(); err != nil {
			return "", err
		} else {
			path = fp.Join(home, suffix)
		}
	}

	if path, err := fp.EvalSymlinks(path); err != nil {
		return "", err
	} else if path, err := fp.Abs(path); err != nil {
		return "", err
	} else {
		return path, nil
	}
}

