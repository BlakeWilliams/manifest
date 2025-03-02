package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var ErrNoPR = errors.New("no PR exists for current branch")
type CommentType int

const (
    ReviewComment CommentType = iota
    FileComment
)

type (
	Client interface {
		DetailsForPull(number int) (*PullRequest, error)
		PullRequestIDsForBranch(sha string) ([]int, error)
		Comment(number int, comment string) error
		Comments(number int) ([]Comment, error)
		ReviewComments(number int) ([]Comment, error)
		FileComment(NewFileComment) error
		ResolveFileComment(comment Comment) error
		ResolveComment(comment Comment) error
		Owner() string
		Repo() string
	}

	defaultClient struct {
		token      string
		owner      string
		repo       string
		HttpClient *http.Client
	}

	// PullRequestFetcher is the interface for ultimately fetching the title and description of a Pull Request
	PullRequestFetcher interface {
		PullsForSha(owner, repo, sha string) ([]int, error)
		PullDetails(owner, repo string, number int) (*PullRequest, error)
	}

	// PullRequest represents a subset of GitHub Pull Request
	PullRequest struct {
		ID    uint
		Title string
		Body  string
		Draft bool
	}
)

func NewClient(token string, owner string, repo string) Client {
	return defaultClient{
		token:      token,
		owner:      owner,
		repo:       repo,
		HttpClient: http.DefaultClient,
	}
}

func (c defaultClient) ReviewComments(number int) ([]Comment, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/comments?per_page=100", c.owner, c.repo, number)
	return c.fetchComments(url, FileComment)
}

func (c defaultClient) Comments(number int) ([]Comment, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments?per_page=100", c.owner, c.repo, number)
	return c.fetchComments(url, ReviewComment)
}

func (c defaultClient) fetchComments(url string, ct CommentType) ([]Comment, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.groot-preview+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}


	var comments []Comment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	for i := range comments {
		comments[i].Type = ct
		comments[i].Stale = true // By default all comments are stale unless we find a matching fingerprint
	}

	return comments, nil
}

func (c defaultClient) DetailsForPull(number int) (*PullRequest, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", c.owner, c.repo, number)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.groot-preview+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	pullRequest := &PullRequest{}
	if err := json.Unmarshal(body, &pullRequest); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return pullRequest, nil
}

func (c defaultClient) PullRequestIDsForBranch(branch string) ([]int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?head=%s:%s", c.owner, c.repo, c.owner, branch)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.groot-preview+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	type pullsForShaResponse struct {
		Number int `json:"number"`
	}

	var pullRequests []pullsForShaResponse
	if err := json.Unmarshal(body, &pullRequests); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	numbers := make([]int, len(pullRequests))
	for i, pull := range pullRequests {
		numbers[i] = pull.Number
	}

	return numbers, nil
}

func (c defaultClient) Comment(number int, comment string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments", c.owner, c.repo, number)
	payload := map[string]string{"body": comment}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

type NewFileComment struct {
	Sha    string
	Number int
	File   string
	Line   int
	Text   string
	Side   string
}

func (c defaultClient) FileComment(fc NewFileComment) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/comments", c.owner, c.repo, fc.Number)
	payload := map[string]interface{}{
		"body":      fc.Text,
		"commit_id": fc.Sha,
		"path":      fc.File,
		"line":      fc.Line,
		"side":      fc.Side,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

func (c defaultClient) ResolveFileComment(comment Comment) error {
	// Update the comment body to strikethrough if not already surrounded with <strike> tags
	if !strings.HasPrefix(comment.Body, "<strike>") && !strings.HasSuffix(comment.Body, "</strike>") {
		comment.Body = fmt.Sprintf("<strike>%s</strike>", comment.Body)
	}

	// Send the updated comment
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/comments/%d", c.owner, c.repo, comment.Id)
	payload := map[string]interface{}{
		"body":     comment.Body,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

func (c defaultClient) ResolveComment(comment Comment) error {
	// Update the comment body to strikethrough if not already surrounded with <strike> tags
	if !strings.HasPrefix(comment.Body, "<strike>") && !strings.HasSuffix(comment.Body, "</strike>") {
		comment.Body = fmt.Sprintf("<strike>%s</strike>", comment.Body)
	}

	// Send the updated comment
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/comments/%d", c.owner, c.repo, comment.Id)
	payload := map[string]interface{}{
		"body": comment.Body,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

func (c defaultClient) Owner() string { return c.owner }
func (c defaultClient) Repo() string  { return c.repo }

type Comment struct {
	Body string `json:"body"`
	Id  int64    `json:"id"`
	Type CommentType `json:"-"`
	Stale bool `json:"-"`
}
