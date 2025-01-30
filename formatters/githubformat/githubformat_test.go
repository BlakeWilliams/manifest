package githubformat

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/blakewilliams/manifest"
	"github.com/blakewilliams/manifest/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeGitHubClient struct {
	mock.Mock
	comments          []github.Comment
	reviewComments    []github.Comment
	fileComments      []github.NewFileComment
	resolvedComments  []github.Comment
	unresolvedComments []github.Comment
}

var _ GitHubClient = (*fakeGitHubClient)(nil)

func (f *fakeGitHubClient) Comment(number int, comment string) error {
	args := f.Called(number, comment)
	f.comments = append(f.comments, github.Comment{Body: comment})
	return args.Error(0)
}

func (f *fakeGitHubClient) FileComment(fc github.NewFileComment) error {
	args := f.Called(fc)
	f.fileComments = append(f.fileComments, fc)
	return args.Error(0)
}

func (f *fakeGitHubClient) Comments(number int) ([]github.Comment, error) {
	args := f.Called(number)
	return f.comments, args.Error(1)
}

func (f *fakeGitHubClient) ReviewComments(number int) ([]github.Comment, error) {
	args := f.Called(number)
	return f.reviewComments, args.Error(1)
}

func (f *fakeGitHubClient) ResolveFileComment(comment github.Comment) error {
	args := f.Called(comment)
	f.resolvedComments = append(f.resolvedComments, comment)
	return args.Error(0)
}

func (f *fakeGitHubClient) ResolveComment(comment github.Comment) error {
	args := f.Called(comment)
	f.resolvedComments = append(f.resolvedComments, comment)
	return args.Error(0)
}

func (f *fakeGitHubClient) UnresolveFileComment(comment github.Comment) error {
	args := f.Called(comment)
	f.unresolvedComments = append(f.unresolvedComments, comment)
	return args.Error(0)
}

func (f *fakeGitHubClient) UnresolveComment(comment github.Comment) error {
	args := f.Called(comment)
	f.unresolvedComments = append(f.unresolvedComments, comment)
	return args.Error(0)
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
	client.On("FileComment", mock.MatchedBy(func(fc github.NewFileComment) bool {
		return fc.Number == 1 &&
			fc.File == "test.go" &&
			fc.Line == 10 &&
			fc.Side == "RIGHT" &&
			strings.Contains(fc.Text, "Test comment") &&
			strings.Contains(fc.Text, "> [!CAUTION]")
	})).Return(nil)

	client.On("Comment", 1, mock.MatchedBy(func(comment string) bool {
		return strings.Contains(comment, "Test comment 2") &&
			strings.Contains(comment, "> [!TIP]")
	})).Return(nil)

	formatter := New(io.Discard, client)
	err := formatter.Format("test", i, result)
	require.NoError(t, err)

	client.AssertExpectations(t)
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
	client.On("FileComment", mock.Anything).Return(fmt.Errorf("comment error"))

	formatter := New(io.Discard, client)
	err := formatter.Format("test", i, result)

	require.Error(t, err)
	require.Equal(t, "comment error", err.Error())

	client.AssertExpectations(t)
}

func TestFormat_Deduplicates(t *testing.T) {
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
			},
			{
				Text:     "File comment!",
				Severity: manifest.SeverityError,
				File:     "test.go",
				Line:     10,
				Side:     "RIGHT",
			},
		},
	}

	client := &fakeGitHubClient{}

	client.On("Comments", 1).Return([]string{"<!-- manifest:test -->", "<!-- manifest:test:test.go:10:RIGHT -->"}, nil)
	client.On("ReviewComments", 1).Return([]string{"<!-- manifest:test:test.go:10:RIGHT -->"}, nil)

	formatter := New(io.Discard, client)
	err := formatter.BeforeAll(i)
	require.NoError(t, err)
	err = formatter.Format("test", i, result)
	require.NoError(t, err)

	client.AssertExpectations(t)
	client.AssertNotCalled(t, "FileComment", mock.Anything)
}

func TestResolveFileComment(t *testing.T) {
	client := &fakeGitHubClient{}
	comment := github.Comment{
		Body: "Test file comment",
		Id:   123,
		Type: github.FileComment,
	}

	client.On("ResolveFileComment", comment).Return(nil)

	err := client.ResolveFileComment(comment)
	require.NoError(t, err)

	client.AssertExpectations(t)
}

func TestResolveComment(t *testing.T) {
	client := &fakeGitHubClient{}
	comment := github.Comment{
		Body: "Test comment",
		Id:   456,
		Type: github.ReviewComment,
	}

	client.On("ResolveComment", comment).Return(nil)

	err := client.ResolveComment(comment)
	require.NoError(t, err)

	client.AssertExpectations(t)
}

func TestUnresolveComment(t *testing.T) {
	client := &fakeGitHubClient{}
	formatter := New(nil, client)

	comment := github.Comment{Body: "~~<!-- manifest:test -->~~", Type: github.ReviewComment}
	client.comments = append(client.comments, comment)

	client.On("UnresolveComment", comment).Return(nil)

	err := formatter.Format("test", &manifest.Import{}, manifest.Result{
		Comments: []manifest.Comment{
			{Text: "Test comment", Severity: manifest.SeverityError},
		},
	})

	assert.NoError(t, err)
	client.AssertCalled(t, "UnresolveComment", comment)
}

func TestUnresolveFileComment(t *testing.T) {
	client := &fakeGitHubClient{}
	formatter := New(nil, client)

	comment := github.Comment{Body: "~~<!-- manifest:test:file:1:side -->~~", Type: github.FileComment}
	client.comments = append(client.comments, comment)

	client.On("UnresolveFileComment", comment).Return(nil)

	err := formatter.Format("test", &manifest.Import{}, manifest.Result{
		Comments: []manifest.Comment{
			{Text: "Test comment", Severity: manifest.SeverityError, File: "file", Line: 1, Side: "side"},
		},
	})

	assert.NoError(t, err)
	client.AssertCalled(t, "UnresolveFileComment", comment)
}
