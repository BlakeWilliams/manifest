package manifest

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Formatter is used to output checker results. For example, you could have
// an stdout formatter for local development and a GitHub formatter to post
// results to a Pull Request.
type Formatter interface {
	Format(source string, i *Import, r Result) error
}

// FormatterWithHooks is just like Formatter, but has hooks for common
// lifecycle events.
type FormatterWithHooks interface {
	BeforeAll(i *Import) error
	AfterAll(i *Import) error

	Formatter
}

type Configuration struct {
	// Concurrency is the number of checkers to run concurrently.
	Concurrency int
	// Formatter is used to output the manifest.Result
	Formatter     Formatter
	Checkers      map[string]string
	FetchPullInfo bool
	// Strict determines if certain checkers or functionality should
	// gracefully degrade based on the environment. e.g. Missing GitHub tokens.
	Strict bool
	// NoGH determines if the token should be pulled from `gh` if
	// MANIFEST_GITHUB_TOKEN is not present.
	NoGH bool
}

type yamlConfiguration struct {
	Manifest struct {
		Concurrency          int    `yaml:"concurrency"`
		Formatter            string `yaml:"formatter"`
		FetchPullRequestInfo bool   `yaml:"fetchPullRequestInfo"`
		NoGH                 bool   `yaml:"noGH"`
		Checkers             map[string]struct {
			Command string `yaml:"command"`
		} `yaml:"checkers"`
	} `yaml:"manifest"`
}

// ParseConfig accepts a reader that should return YAML configuration for
// manifest. It returns the parsed configuration.
func ParseConfig(r io.Reader, c *Configuration, formatters map[string]Formatter) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("could not read configuration file: %w", err)
	}

	var yamlConfig yamlConfiguration
	err = yaml.Unmarshal(content, &yamlConfig)
	if err != nil {
		return fmt.Errorf("could not parse configuration file: %w", err)
	}

	if yamlConfig.Manifest.Concurrency > 0 {
		c.Concurrency = yamlConfig.Manifest.Concurrency
	}

	if yamlConfig.Manifest.FetchPullRequestInfo {

		c.FetchPullInfo = true
	}

	if yamlConfig.Manifest.Formatter != "" {
		formatter, ok := formatters[yamlConfig.Manifest.Formatter]
		if !ok {
			return fmt.Errorf("could not find formatter '%s'", yamlConfig.Manifest.Formatter)
		}
		c.Formatter = formatter
	}

	if c.Checkers == nil {
		c.Checkers = make(map[string]string, len(yamlConfig.Manifest.Checkers))
	}
	for name, checker := range yamlConfig.Manifest.Checkers {
		c.Checkers[name] = checker.Command
	}

	return nil
}
