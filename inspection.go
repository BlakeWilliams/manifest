package manifest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/blakewilliams/manifest/github"
	"github.com/blakewilliams/manifest/pkg/multierror"
	"golang.org/x/sync/errgroup"
)

var ErrCheckReportedError = errors.New("one or more checkers reported an error")

type Check struct {
	config *Configuration
	Import *Import
}

func NewCheck(c *Configuration, diffReader io.Reader) (*Check, error) {
	diff, err := NewDiff(diffReader)
	if err != nil {
		return nil, fmt.Errorf("could not create diff: %w", err)
	}

	check := &Check{
		config: c,
		Import: &Import{Strict: c.Strict, Diff: diff},
	}

	return check, nil
}

func (i *Check) PopulatePullDetails(gh github.Client, sha string, prNum int) error {
	pr, err := gh.DetailsForPull(prNum)
	if err != nil {
		return err
	}

	i.Import.Pull = &Pull{
		RepoOwner:   gh.Owner(),
		RepoName:    gh.Repo(),
		Number:      prNum,
		Title:       pr.Title,
		Description: pr.Body,
		Draft:       pr.Draft,
	}

	i.Import.CurrentSha = sha

	return nil
}

func (i *Check) ImportJSON() ([]byte, error) {
	out, err := json.Marshal(i.Import)
	if err != nil {
		return nil, fmt.Errorf("could not marshall output for import JSON: %w", err)
	}

	return out, nil
}

// Perform accepts a configuration and a diff, then runs + reports on the rules
// based on the configuration+output.
func (i *Check) Perform() error {
	importJSON, err := i.ImportJSON()
	if err != nil {
		return err
	}

	// TODO add a timout config
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(i.config.Concurrency)

	if f, ok := i.config.Formatter.(FormatterWithHooks); ok {
		err := f.BeforeAll(i.Import)
		if err != nil {
			return fmt.Errorf("formatter before all hook failed: %w", err)
		}

		// TODO handle err
		defer f.AfterAll(i.Import)
	}

	var wg sync.WaitGroup
	multiErr := &multierror.Error{}

	hasCheckErrors := false

	for name, check := range i.config.Checkers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}

			cmd := exec.Command("sh", "-c", check)
			cmd.Stdin = bytes.NewReader(importJSON)
			output, err := cmd.Output()
			if err != nil {
				multiErr.Add(fmt.Errorf("`%s` check failed to run: %w", name, err))
				fmt.Fprint(os.Stderr, string(output))
				return
			}

			var result Result
			err = json.Unmarshal(output, &result)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse output for check %s: %s\n", name, err)
				fmt.Fprint(os.Stderr, string(output))
				multiErr.Add(err)
				return
			}

			if result.Failure != "" {
				multiErr.Add(fmt.Errorf("Check %s failed with reported reason: %s", name, result.Failure))
				return
			}

			if !hasCheckErrors {
				for _, comment := range result.Comments {
					if comment.Severity == SeverityError {
						hasCheckErrors = true
						break
					}
				}
			}

			if err := i.config.Formatter.Format(name, i.Import, result); err != nil {
				multiErr.Add(err)
				return
			}
		}()
	}

	wg.Wait()

	if multiErr.None() {
		if hasCheckErrors {
			return ErrCheckReportedError
		}

		return nil
	}

	return multiErr.ErrorOrNil()
}
