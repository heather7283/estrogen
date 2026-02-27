package main

import (
	"fmt"
	"os"
	"path/filepath"
	re "regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type FilterType int

const (
	FilterTypeInclude FilterType = iota
	FilterTypeExclude FilterType = iota
)

type Filter struct {
	Type FilterType
	Regex *re.Regexp
}

func (f *Filter) UnmarshalTOML(_data any) error {
	data, ok := _data.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected filter value type: %T", _data)
	} else if len(data) != 1 {
		return fmt.Errorf("exactly one of include, exclude must be specified")
	}

	var (
		filterType FilterType
		regexStr string
	)
	if v, ok := data["include"]; ok {
		filterType = FilterTypeInclude
		regexStr, ok = v.(string)
		if !ok {
			return fmt.Errorf("unexpected include value type: expected string, got %T", v)
		}
	} else if v, ok := data["exclude"]; ok {
		filterType = FilterTypeExclude
		regexStr, ok = v.(string)
		if !ok {
			return fmt.Errorf("unexpected exclude value type: expected string, got %T", v)
		}
	} else {
		return fmt.Errorf("exactly one of include, exclude must be specified")
	}

	filterRegex, err := re.Compile(regexStr)
	if err != nil {
		return err
	}

	*f = Filter{
		Type: filterType,
		Regex: filterRegex,
	}

	return nil
}

type Rename struct {
	Pattern *re.Regexp
	Replacement string
}

func (r *Rename) UnmarshalTOML(_data any) error {
	data, ok := _data.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected rename value type: %T", _data)
	} else if len(data) > 2 {
		return fmt.Errorf("too many keys, expected pattern and replacement")
	}

	if _pattern, ok := data["pattern"]; !ok {
		return fmt.Errorf("pattern not found")
	} else if pattern, ok := _pattern.(string); !ok {
		return fmt.Errorf("pattern should be a string")
	} else if re, err := re.Compile(pattern); err != nil {
		return err
	} else {
		r.Pattern = re
	}

	if _replacement, ok := data["replacement"]; !ok {
		return fmt.Errorf("replacement not found")
	} else if replacement, ok := _replacement.(string); !ok {
		return fmt.Errorf("replacement should be a string")
	} else {
		r.Replacement = replacement
	}

	return nil
}

type Rule struct {
	Src *re.Regexp
	Dst string
	Cmd []string
}

func (r *Rule) UnmarshalTOML(_data any) error {
	data, ok := _data.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected rename value type: %T", _data)
	} else if len(data) > 3 {
		return fmt.Errorf("too many keys, expected src, dst and cmd")
	}

	if _src, ok := data["src"]; !ok {
		return fmt.Errorf("src not found")
	} else if src, ok := _src.(string); !ok {
		return fmt.Errorf("src should be a string")
	} else if re, err := re.Compile(src); err != nil {
		return err
	} else {
		r.Src = re
	}

	if _dst, ok := data["dst"]; !ok {
		return fmt.Errorf("dst not found")
	} else if dst, ok := _dst.(string); !ok {
		return fmt.Errorf("dst should be a string")
	} else {
		r.Dst = dst
	}

	if _cmd, ok := data["cmd"]; !ok {
		return fmt.Errorf("cmd not found")
	} else if cmdStr, ok := _cmd.(string); ok {
		r.Cmd = []string{"/bin/sh", "-c", cmdStr, "sh", "@SRC@", "@DST@"}
	} else if cmdArr, ok := _cmd.([]any); ok {
		cmd := make([]string, len(cmdArr))
		for i, e := range cmdArr {
			if str, ok := e.(string); ok {
				cmd[i] = str
			} else {
				return fmt.Errorf("cmd array elements should be strings, got %T at %d", e, i)
			}
		}
	} else {
		return fmt.Errorf("cmd should be either string or array of strings, got %T", _cmd)
	}

	return nil
}

type Settings struct {
	DeleteRemoved bool `toml:"delete_removed"`
	CopyUnmatched bool `toml:"copy_unmatched"`
	ExcludeByDefault bool `toml:"exclude_by_default"`
}

type Config struct {
	Src, Dst string
	Settings Settings
	Filters []Filter `toml:"filter"`
	Renames []Rename `toml:"rename"`
	Rules []Rule `toml:"rule"`
}

func ExpandHome(path *string) error {
	if suffix, hasPrefix := strings.CutPrefix(*path, "~/"); hasPrefix {
		if home, err := os.UserHomeDir(); err != nil {
			return err
		} else {
			*path = filepath.Join(home, suffix)
		}
	}

	return nil
}

func ParseConfig(path string) (*Config, error) {
	config := Config{
		Dst: ".",
		Settings: Settings{
			DeleteRemoved: false,
			CopyUnmatched: true,
		},
	}
	if md, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	} else if undecoded := md.Undecoded(); len(undecoded) > 0 {
		return nil, fmt.Errorf("unknown key: %s", undecoded[0])
	}

	if err := ExpandHome(&config.Src); err != nil {
		return nil, fmt.Errorf("failed to expand src: %v", err)
	}
	if err := ExpandHome(&config.Dst); err != nil {
		return nil, fmt.Errorf("failed to expand dst: %v", err)
	}

	if suffix, hasPrefix := strings.CutPrefix(config.Src, "~/"); hasPrefix {
		if home, err := os.UserHomeDir(); err != nil {
			return nil, fmt.Errorf("failed to expand src: %v", err)
		} else {
			config.Src = filepath.Join(home, suffix)
		}
	}

	return &config, nil
}

