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
		argv = append(argv, "/bin/sh", val, "sh", "@SRC@", "@DST@")
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
	// Include and Exclude are mutually exclusive
	Exclude, Include *string
	Type FilterType `toml:"-"`
	Re *re.Regexp `toml:"-"`
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
	Rules []Rule `toml:"rule"`
}

func ParseConfig() (*Config, error) {
	path := "./estrogen.toml"

	config := Config{
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

	if suffix, hasPrefix := strings.CutPrefix(config.Src, "~/"); hasPrefix {
		if home, err := os.UserHomeDir(); err != nil {
			return nil, fmt.Errorf("failed to expand src: %v", err)
		} else {
			config.Src = filepath.Join(home, suffix)
		}
	}

	for i := range config.Filters {
		f := &config.Filters[i]
		hasInclude := f.Include != nil
		hasExclude := f.Exclude != nil
		if !((hasInclude && !hasExclude) || (!hasInclude && hasExclude)) {
			return nil, fmt.Errorf("filter %d: both include and exclude patterns specified", i + 1)
		} else if hasInclude {
			if len(*f.Include) < 1 {
				return nil, fmt.Errorf("filter %d: empty include pattern", i + 1)
			} else if regex, err := re.Compile(*f.Include); err != nil {
				return nil, fmt.Errorf("filter %d: %v", i + 1, err)
			} else {
				f.Type = FilterTypeInclude
				f.Re = regex
			}
		} else if hasExclude {
			if len(*f.Exclude) < 1 {
				return nil, fmt.Errorf("filter %d: empty exclude pattern", i + 1)
			} else if regex, err := re.Compile(*f.Exclude); err != nil {
				return nil, fmt.Errorf("filter %d: %v", i + 1, err)
			} else {
				f.Type = FilterTypeExclude
				f.Re = regex
			}
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

