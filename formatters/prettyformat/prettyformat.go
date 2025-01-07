package prettyformat

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/fatih/color"

	"github.com/blakewilliams/manifest"
)

type Formatter struct {
	out io.Writer
	mu  sync.Mutex
}

var warnColor = color.New(color.FgYellow, color.Bold)
var errorColor = color.New(color.FgRed, color.Bold)
var infoColor = color.New(color.FgBlue, color.Bold)

func New(out io.Writer) *Formatter {
	return &Formatter{out: out}
}

func (s *Formatter) Format(source string, i *manifest.Import, r manifest.Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, comment := range r.Comments {
		switch comment.Severity {
		case manifest.SeverityError:
			errorColor.Fprintf(s.out, "== Error: %s\n", source)
			if comment.File != "" && comment.Line != 0 {
				errorColor.Fprintf(s.out, "%s:%d\n", comment.File, comment.Line)
			}
		case manifest.SeverityWarn:
			warnColor.Fprintf(s.out, "== Warning: %s\n", source)
			if comment.File != "" && comment.Line != 0 {
				warnColor.Fprintf(s.out, "%s:%d\n", comment.File, comment.Line)
			}
		case manifest.SeverityInfo:
			warnColor.Fprintf(s.out, "== Info: %s\n", source)
			if comment.File != "" && comment.Line != 0 {
				infoColor.Fprintf(s.out, "%s:%d\n", comment.File, comment.Line)
			}
		}

		for _, line := range strings.Split(comment.Text, "\n") {
			fmt.Fprintf(s.out, "  > %s\n", line)
		}

		fmt.Printf("\n\n")
	}

	return nil
}
