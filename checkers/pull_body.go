package checkers

import (
	"strings"

	"github.com/blakewilliams/manifest"
)

func PullBody(entry *manifest.Import, r *manifest.Result) error {
	if entry.Pull.Title == "" && entry.Pull.Description == "" && entry.Strict {
		r.Failure = "No pull request description provided"
	}

	if strings.TrimSpace(entry.Pull.Description) == "" {
		r.Error("It looks like your pull request description is empty! Please provide a description of your changes.")
	}
	return nil
}
