package manifest

// Result is the result of a rule being run against a diff. Manifest uses the
// result to determine if the PR passes and where to comment if configured to.
type Result struct {
	Failure  string    `json:"failure,omitempty"`
	Comments []Comment `json:"comments"`
}

type Severity string

const (
	// SeverityInfo does not fail the build and does not emhphasize the message.
	SeverityInfo Severity = "Info"
	// SeverityWarn does not fail the build, but emphasizes caution.
	SeverityWarn Severity = "Warn"
	// SeverityError fails the build
	SeverityError Severity = "Error"
)

// Comment is a comment that can be left on a PR or left as a warning in the
// terminal.
type Comment struct {
	// The file to comment on. Leave blank if the comment should be made in the
	// PR.
	File string `json:"file"`
	// The line to comment on. Leave blank alongside the File field to comment
	// top-level.
	Line uint `json:"line"`
	// Side is the side of the diff to comment on. Can be "LEFT" or "RIGHT".
	// This is required for file comments.
	Side string `json:"side"`
	// The text to include in your comment.
	Text string `json:"text"`
	// Severity of the comment. Defaults to Info.
	Severity Severity `json:"severity"`
}

// Warn adds a general warning that will be shown to the user based on the
// provided formatter.
func (r *Result) Warn(message string) {
	r.Comments = append(r.Comments, Comment{
		Text:     message,
		Severity: SeverityWarn,
	})
}

// WarnLine adds a warning to a specific line in a file that will be shown to the
// user based on the provided formatter.
func (r *Result) WarnLine(file string, side string, line uint, message string) {
	r.Comments = append(r.Comments, Comment{
		File:     file,
		Line:     line,
		Text:     message,
		Side:     side,
		Severity: SeverityWarn,
	})
}

// Error adds a general warning that will be shown to the user based on the
// provided formatter.
func (r *Result) Error(message string) {
	r.Comments = append(r.Comments, Comment{
		Text:     message,
		Severity: SeverityError,
	})
}

// ErrorLine adds a warning to a specific line in a file that will be shown to the
// user based on the provided formatter.
func (r *Result) ErrorLine(file string, side string, line uint, message string) {
	r.Comments = append(r.Comments, Comment{
		File:     file,
		Side:     side,
		Line:     line,
		Text:     message,
		Severity: SeverityError,
	})
}
