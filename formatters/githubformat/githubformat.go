package githubformat

import (
	"fmt"
	"strings"

	"github.com/blakewilliams/manifest"
	"github.com/blakewilliams/manifest/github"
)

var footer = "\n\n<sub>This comment was generated by the `%s` inspector using [manifest](https://github.com/blakewilliams/manifest)</sup>"

type Formatter struct {
	client github.Client
	number int
	sha    string
}

func New(client github.Client, number int, sha string) *Formatter {
	return &Formatter{
		client: client,
		number: number,
		sha:    sha,
	}
}

func (f *Formatter) Format(source string, i *manifest.Import, r manifest.Result) error {
	var topLevelmessage strings.Builder

	for _, comment := range r.Comments {
		var message strings.Builder
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

			c := github.FileComment{
				Sha:    f.sha,
				Text:   message.String(),
				Number: f.number,
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

		if err := f.client.Comment(i.PullNumber, topLevelmessage.String()); err != nil {
			return err
		}

		fmt.Printf("Commenting on PR:\n %s\n", topLevelmessage.String())
	}

	return nil
}