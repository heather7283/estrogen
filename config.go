package main

import (
	"fmt"
	"os"
	"path/filepath"
	re "regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type Command []string

func (c *Command) UnmarshalTOML(data any) error {
	var argv []string

	switch val := data.(type) {
	case string:
		argv = append(argv, "/bin/sh", "-c", val, "sh", "@SRC@", "@DST@")
	case []any:
		for i, v := range val {
			if str, isStr := v.(string); isStr {
				argv = append(argv, str)
			} else {
				return fmt.Errorf("cmd array elements should be strings, got %T at %d", v, i)
			}
		}
	default:
		return fmt.Errorf("cmd should be either string or array of string, got %T", val)
	}

	*c = argv
	return nil
}

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
	Src string
	SrcRe *re.Regexp `toml:"-"`
	Dst string
	Cmd Command
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

	for i := range config.Rules {
		r := &config.Rules[i]
		if regex, err := re.Compile(r.Src); err != nil {
			return nil, fmt.Errorf("rule %d: failed to compile src regex: %v", i + 1, err)
		} else if len(r.Dst) < 1 {
			return nil, fmt.Errorf("rule %d: empty dst pattern", i + 1)
		} else if len(r.Cmd) < 1 {
			return nil, fmt.Errorf("rule %d: empty command", i + 1)
		} else {
			r.SrcRe = regex
		}
	}

	return &config, nil
}

