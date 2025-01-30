package githubformat

import (
	"io"
	"testing"

	"github.com/blakewilliams/manifest"
	"github.com/blakewilliams/manifest/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGitHubClient struct {
	comments          []github.Comment
	reviewComments    []github.Comment
	fileComments      []github.NewFileComment
	resolvedComments  []github.Comment
	unresolvedComments []github.Comment
}

var _ GitHubClient = (*fakeGitHubClient)(nil)

func (f *fakeGitHubClient) Comment(number int, comment string) error {
	f.comments = append(f.comments, github.Comment{Body: comment})
	return nil
}

func (f *fakeGitHubClient) FileComment(fc github.NewFileComment) error {
	f.fileComments = append(f.fileComments, fc)
	return nil
}

func (f *fakeGitHubClient) Comments(number int) ([]github.Comment, error) {
	return f.comments, nil
}

func (f *fakeGitHubClient) ReviewComments(number int) ([]github.Comment, error) {
	return f.reviewComments, nil
}

func (f *fakeGitHubClient) ResolveFileComment(comment github.Comment) error {
	f.resolvedComments = append(f.resolvedComments, comment)
	return nil
}

func (f *fakeGitHubClient) ResolveComment(comment github.Comment) error {
	f.resolvedComments = append(f.resolvedComments, comment)
	return nil
}

func (f *fakeGitHubClient) UnresolveFileComment(comment github.Comment) error {
	f.unresolvedComments = append(f.unresolvedComments, comment)
	return nil
}

func (f *fakeGitHubClient) UnresolveComment(comment github.Comment) error {
	f.unresolvedComments = append(f.unresolvedComments, comment)
	return nil
}

func TestFormat_FileComment(t *testing.T) {
	i := &manifest.Import{
		Pull: &manifest.Pull{
			Number: 1,
		},
	}

	result := manifest.Result{
		Comments: []manifest.Comment{
			{
				Text:     "Test comment",
				Severity: manifest.SeverityError,
				File:     "test.go",
				Line:     10,
				Side:     "RIGHT",
			},
			{
				Text:     "Test comment 2",
				Severity: manifest.SeverityInfo,
			},
		},
	}

	client := &fakeGitHubClient{}
	formatter := New(io.Discard, client)
	err := formatter.Format("test", i, result)
	require.NoError(t, err)

	require.Len(t, client.fileComments, 1)
	require.Contains(t, client.fileComments[0].Text, "Test comment")
	require.Contains(t, client.fileComments[0].Text, "> [!CAUTION]")

	require.Len(t, client.comments, 1)
	require.Contains(t, client.comments[0].Body, "Test comment 2")
	require.Contains(t, client.comments[0].Body, "> [!TIP]")
}

func TestFormat_CommentError(t *testing.T) {
	i := &manifest.Import{
		Pull: &manifest.Pull{
			Number: 1,
		},
	}

	result := manifest.Result{
		Comments: []manifest.Comment{
			{
				Text:     "Test comment",
				Severity: manifest.SeverityError,
				File:     "test.go",
				Line:     10,
				Side:     "RIGHT",
			},
		},
	}

	client := &fakeGitHubClient{}
	formatter := New(io.Discard, client)
	err := formatter.Format("test", i, result)

	require.NoError(t, err)
	require.Len(t, client.fileComments, 1)
}

func TestResolveFileComment(t *testing.T) {
	client := &fakeGitHubClient{}
	formatter := New(io.Discard, client)

	comment := github.Comment{Body: "<!-- manifest:test:file:1:side -->", Type: github.FileComment, Stale: true}
	client.comments = append(client.comments, comment)

	i := &manifest.Import{
		Pull: &manifest.Pull{
			Number: 1,
		},
	}

	err := formatter.BeforeAll(i)
	require.NoError(t, err)

	err = formatter.Format("test", i, manifest.Result{
		Comments: []manifest.Comment{
			{Text: "Another comment", Severity: manifest.SeverityError, File: "file", Line: 1, Side: "side"},
		},
	})

	err = formatter.AfterAll(i)

	assert.NoError(t, err)
	require.Len(t, client.resolvedComments, 1)
	require.Equal(t, comment.Body, client.resolvedComments[0].Body)
}

func TestResolveComment(t *testing.T) {
	client := &fakeGitHubClient{}
	formatter := New(io.Discard, client)

	comment := github.Comment{Body: "<!-- manifest:test -->", Type: github.ReviewComment, Stale: true}
	client.comments = append(client.comments, comment)

	i := &manifest.Import{
		Pull: &manifest.Pull{
			Number: 1,
		},
	}

	err := formatter.BeforeAll(i)
	require.NoError(t, err)

	err = formatter.Format("test", i, manifest.Result{
		Comments: []manifest.Comment{
			{Text: "Another comment", Severity: manifest.SeverityError},
		},
	})

	err = formatter.AfterAll(i)

	assert.NoError(t, err)
	require.Len(t, client.resolvedComments, 1)
	require.Equal(t, comment.Body, client.resolvedComments[0].Body)
}

func TestUnresolveComment(t *testing.T) {
	client := &fakeGitHubClient{}
	formatter := New(io.Discard, client)

	comment := github.Comment{Body: "~~<!-- manifest:test -->~~", Type: github.ReviewComment}
	client.comments = append(client.comments, comment)

	i := &manifest.Import{
		Pull: &manifest.Pull{
			Number: 1,
		},
	}

	err := formatter.BeforeAll(i)
	require.NoError(t, err)

	err = formatter.Format("test", i, manifest.Result{
		Comments: []manifest.Comment{
			{Text: "Test comment", Severity: manifest.SeverityError},
		},
	})

	assert.NoError(t, err)
	require.Len(t, client.unresolvedComments, 1)
	require.Equal(t, comment.Body, client.unresolvedComments[0].Body)
}

func TestUnresolveFileComment(t *testing.T) {
	client := &fakeGitHubClient{}
	formatter := New(io.Discard, client)

	comment := github.Comment{Body: "~~<!-- manifest:test:file:1:side -->~~", Type: github.FileComment}
	client.comments = append(client.comments, comment)

	i := &manifest.Import{
		Pull: &manifest.Pull{
			Number: 1,
		},
	}

	err := formatter.BeforeAll(i)
	require.NoError(t, err)

	err = formatter.Format("test", i, manifest.Result{
		Comments: []manifest.Comment{
			{Text: "Test comment", Severity: manifest.SeverityError, File: "file", Line: 1, Side: "side"},
		},
	})

	assert.NoError(t, err)
	require.Len(t, client.unresolvedComments, 1)
	require.Equal(t, comment, client.unresolvedComments[0])
}
