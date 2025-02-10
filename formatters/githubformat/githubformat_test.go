package githubformat

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/blakewilliams/manifest"
	"github.com/blakewilliams/manifest/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeGitHubClient struct {
	mock.Mock
}

var _ GitHubClient = (*fakeGitHubClient)(nil)

func (f *fakeGitHubClient) Comment(number int, comment string) error {
	args := f.Called(number, comment)
	return args.Error(0)
}

func (f *fakeGitHubClient) FileComment(fc github.NewFileComment) error {
	args := f.Called(fc)
	return args.Error(0)
}

func (f *fakeGitHubClient) Comments(number int) ([]github.Comment, error) {
	args := f.Called(number)
	return args.Get(0).([]github.Comment), args.Error(1)
}

func (f *fakeGitHubClient) ReviewComments(number int) ([]github.Comment, error) {
	args := f.Called(number)
	return args.Get(0).([]github.Comment), args.Error(1)
}

func (f *fakeGitHubClient) ResolveFileComment(comment github.Comment) error {
	args := f.Called(comment)
	return args.Error(0)
}

func (f *fakeGitHubClient) ResolveComment(comment github.Comment) error {
	args := f.Called(comment)
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

	client.On("Comments", 1).Return([]github.Comment{
		{Body: "<!-- manifest:test -->", Type: github.ReviewComment},
		{Body: "<!-- manifest:test:test.go:10:RIGHT -->", Type: github.FileComment},
	}, nil)
	client.On("ReviewComments", 1).Return([]github.Comment{
		{Body: "<!-- manifest:test:test.go:10:RIGHT -->", Type: github.FileComment},
	}, nil)

	formatter := New(io.Discard, client)
	err := formatter.BeforeAll(i)
	require.NoError(t, err)
	err = formatter.Format("test", i, result)
	require.NoError(t, err)

	client.AssertExpectations(t)

	client.AssertNotCalled(t, "FileComment", mock.Anything)
}

func TestFormat_ResolveComment(t *testing.T) {
	i := &manifest.Import{
		Pull: &manifest.Pull{
			Number: 1,
		},
	}

	result := manifest.Result{
		Comments: []manifest.Comment{},
	}

	client := &fakeGitHubClient{}

	client.On("Comments", 1).Return([]github.Comment{
		{Body: "<!-- manifest:test -->", Type: github.ReviewComment, Stale: true},
	}, nil)
	client.On("ReviewComments", 1).Return([]github.Comment{}, nil)
	client.On("ResolveComment", mock.Anything).Return(nil)

	formatter := New(io.Discard, client)
	err := formatter.BeforeAll(i)
	require.NoError(t, err)


	err = formatter.Format("test", i, result)
	require.NoError(t, err)
	err = formatter.AfterAll(i)
	require.NoError(t, err)

	client.AssertExpectations(t)
	client.AssertCalled(t, "ResolveComment", mock.Anything)
}
