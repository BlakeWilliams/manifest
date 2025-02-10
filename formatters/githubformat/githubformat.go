package githubformat

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/blakewilliams/manifest"
	"github.com/blakewilliams/manifest/formatters/prettyformat"
	"github.com/blakewilliams/manifest/github"
)

var footer = "\n\n<sub>This comment was generated by the `%s` checker using [manifest](https://github.com/blakewilliams/manifest)</sup>"

type Formatter struct {
	client           GitHubClient
	existingComments map[string]github.Comment
	cliFormatter     manifest.Formatter
}

var _ manifest.FormatterWithHooks = (*Formatter)(nil)

type GitHubClient interface {
	Comment(number int, comment string) error
	Comments(number int) ([]github.Comment, error)
	ReviewComments(number int) ([]github.Comment, error)
	FileComment(github.NewFileComment) error
	ResolveFileComment(comment github.Comment) error
	ResolveComment(comment github.Comment) error
}

// TODO remove number and sha, use the import instead
func New(out io.Writer, client GitHubClient) *Formatter {
	return &Formatter{
		client:           client,
		existingComments: make(map[string]github.Comment),
		cliFormatter:     prettyformat.New(out),
	}
}

var fingerprintRegex = regexp.MustCompile(`<!--\s*(manifest:.*?)\s*-->`)

// BeforeAll grabs the comments in the PR so it can attempt to de-duplicat
// them.
func (f *Formatter) BeforeAll(i *manifest.Import) error {
	comments, err := f.client.Comments(i.Pull.Number)
	if err != nil {
		return err
	}

	for _, comment := range comments {
		// Ignore any comments that were previously resolved. New ones will be created if necessary
		if strings.HasPrefix(comment.Body, "<strike>") {
			continue;
		}
		matches := fingerprintRegex.FindAllStringSubmatch(comment.Body, -1)
		for _, fingerprint := range matches {
			f.existingComments[fingerprint[1]] = comment
		}
	}

	comments, err = f.client.ReviewComments(i.Pull.Number)
	if err != nil {
		return err
	}

	for _, comment := range comments {
		matches := fingerprintRegex.FindAllStringSubmatch(comment.Body, -1)
		for _, fingerprint := range matches {
			f.existingComments[fingerprint[1]] = comment
		}
	}
	return nil
}

func (f *Formatter) AfterAll(i *manifest.Import) error {

	// cleanup stale comments
	// If the comment is a file comment resolve it, otherwise update the text formatting to strike through
	for _, comment := range f.existingComments {
		if !comment.Stale {
			continue
		}

		if comment.Type == github.FileComment {
			f.client.ResolveFileComment(comment)
		} else {
			f.client.ResolveComment(comment)
		}
	}

	return nil
}

func (f *Formatter) Format(source string, i *manifest.Import, r manifest.Result) error {
	var topLevelmessage strings.Builder

	for _, comment := range r.Comments {
		fingerprint := fingerprint(source, comment)
		if ec, ok := f.existingComments[fingerprint]; ok {
			// This comment won't be marked as resolved because the checker still thinks it's a problem.
			ec.Stale = false
			continue
		}

		var message strings.Builder

		message.WriteString(fmt.Sprintf("<!-- %s -->\n\n", fingerprint))

		switch comment.Severity {
		case manifest.SeverityError:
			message.WriteString("> [!CAUTION]\n")
		case manifest.SeverityWarn:
			message.WriteString("> [!WARNING]\n")
		case manifest.SeverityInfo:
			message.WriteString("> [!TIP]\n")
		}

		if comment.File != "" && comment.Line != 0 {
			for _, s := range strings.Split(comment.Text, "\n") {
				message.WriteString("> ")
				message.WriteString(s)
				message.WriteString("\n")
			}

			message.WriteString(fmt.Sprintf(footer, source))

			c := github.NewFileComment{
				Sha:    i.CurrentSha,
				Text:   message.String(),
				Number: i.Pull.Number,
				File:   comment.File,
				Line:   int(comment.Line),
				Side:   comment.Side,
			}
			if err := f.client.FileComment(c); err != nil {
				return err
			}
		} else {
			for _, s := range strings.Split(comment.Text, "\n") {
				message.WriteString("> ")
				message.WriteString(s)
				message.WriteString("\n")
			}

			message.WriteString("\n\n")
			topLevelmessage.WriteString(message.String())
		}
	}

	if topLevelmessage.Len() > 0 {
		topLevelmessage.WriteString(fmt.Sprintf(footer, source))

		if err := f.client.Comment(i.Pull.Number, topLevelmessage.String()); err != nil {
			return err
		}

		fmt.Printf("Commenting on PR:\n %s\n", topLevelmessage.String())
	}

	return f.cliFormatter.Format(source, i, r)
}

func fingerprint(source string, comment manifest.Comment) string {
	if comment.File == "" || comment.Line == 0 {
		return fmt.Sprintf("manifest:%s", source)
	}

	// TODO this should not use line number exactly, but hacky WIP
	// track via hunk position, too?
	return fmt.Sprintf("manifest:%s:%s:%d:%s", source, comment.File, comment.Line, comment.Side)
}
