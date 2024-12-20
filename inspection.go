package manifest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/blakewilliams/manifest/github"
	"golang.org/x/sync/errgroup"
)

type Inspection struct {
	config *Configuration
	Import *Import
}

func NewInspection(c *Configuration, diffReader io.Reader) (*Inspection, error) {
	diff, err := NewDiff(diffReader)
	if err != nil {
		return nil, fmt.Errorf("could not create diff: %w", err)
	}

	inspection := &Inspection{
		config: c,
		Import: &Import{Strict: c.Strict, Diff: diff},
	}

	return inspection, nil
}

func (i *Inspection) PopulatePullDetails(gh github.Client, sha string, prNum int) error {
	pr, err := gh.DetailsForPull(prNum)
	if err != nil {
		return err
	}

	i.Import.RepoOwner = gh.Owner()
	i.Import.RepoName = gh.Repo()
	i.Import.PullNumber = prNum

	i.Import.PullTitle = pr.Title
	i.Import.PullDescription = pr.Body

	i.Import.CurrentSha = sha

	return nil
}

func (i *Inspection) ImportJSON() ([]byte, error) {
	out, err := json.Marshal(i.Import)
	if err != nil {
		return nil, fmt.Errorf("could not marshall output for import JSON: %w", err)
	}

	return out, nil
}

// Inspect accepts a configuration and a diff, then runs + reports on the rules
// based on the configuration+output.
func (i *Inspection) Perform() error {
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

	for name, inspector := range i.config.Inspectors {
		g.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}

			cmd := exec.Command("sh", "-c", inspector)
			cmd.Stdin = bytes.NewReader(importJSON)
			output, err := cmd.Output()
			if err != nil {
				return err
			}

			var result Result
			err = json.Unmarshal(output, &result)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse output for inspector %s: %s\n", name, err)
				return err
			}

			if result.Failure != "" {
				return fmt.Errorf("inspector %s failed with reported reason: %s", name, result.Failure)
			}

			for _, comment := range result.Comments {
				if comment.Severity == SeverityError {
					break
				}
			}

			return i.config.Formatter.Format(name, i.Import, result)
		})
	}

	err = g.Wait()
	if err != nil {
		return fmt.Errorf("one or more rules failed: %w", err)
	}

	return nil
}
