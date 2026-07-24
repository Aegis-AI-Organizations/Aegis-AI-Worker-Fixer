package fixer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LocalGitProvider struct {
	PRProvider GitProvider
}

func (p *LocalGitProvider) Clone(ctx context.Context, repo Repository, baseBranch string) (string, error) {
	if strings.TrimSpace(repo.CloneURL) == "" {
		return "", errors.New("repository clone_url is required")
	}
	dir, err := os.MkdirTemp("", "aegis-fixer-*")
	if err != nil {
		return "", err
	}
	args := []string{"clone", "--branch", baseBranch, repo.CloneURL, dir}
	if out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return dir, nil
}

func (p *LocalGitProvider) CreateBranch(ctx context.Context, workspacePath string, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", "-B", branch)
	cmd.Dir = workspacePath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *LocalGitProvider) CommitFiles(ctx context.Context, req CommitRequest) error {
	for _, file := range req.Files {
		cmd := exec.CommandContext(ctx, "git", "add", file)
		cmd.Dir = filepath.Dir(file)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git add: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	workspace := commonWorkspace(req.Files)
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", req.Message)
	cmd.Dir = workspace
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, strings.TrimSpace(string(out)))
	}
	cmd = exec.CommandContext(ctx, "git", "push", "-u", "origin", req.Branch)
	cmd.Dir = workspace
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *LocalGitProvider) OpenPullRequest(ctx context.Context, req PullRequestRequest) (string, error) {
	if p.PRProvider == nil {
		return "", errors.New("pull request provider is required")
	}
	return p.PRProvider.OpenPullRequest(ctx, req)
}

func commonWorkspace(files []string) string {
	if len(files) == 0 {
		return "."
	}
	return filepath.Dir(files[0])
}

type GitHubProvider struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

type GitLabProvider struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

func NewGitProviderFromEnv(client *http.Client) (GitProvider, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return &GitHubProvider{Token: token, BaseURL: envDefault("GITHUB_API_URL", "https://api.github.com"), Client: client}, nil
	}
	if token := strings.TrimSpace(os.Getenv("GITLAB_TOKEN")); token != "" {
		return &GitLabProvider{Token: token, BaseURL: envDefault("GITLAB_API_URL", "https://gitlab.com"), Client: client}, nil
	}
	return nil, errors.New("GITHUB_TOKEN or GITLAB_TOKEN is required")
}

func (p *GitHubProvider) Clone(context.Context, Repository, string) (string, error) {
	return "", errors.New("GitHubProvider only supports pull request API operations")
}

func (p *GitHubProvider) CreateBranch(context.Context, string, string) error {
	return errors.New("GitHubProvider only supports pull request API operations")
}

func (p *GitHubProvider) CommitFiles(context.Context, CommitRequest) error {
	return errors.New("GitHubProvider only supports pull request API operations")
}

func (p *GitHubProvider) OpenPullRequest(ctx context.Context, req PullRequestRequest) (string, error) {
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/repos/" + req.Repository.Owner + "/" + req.Repository.Name + "/pulls"
	payload := map[string]string{"title": req.Title, "head": req.HeadBranch, "base": req.BaseBranch, "body": req.Body}
	var response struct {
		HTMLURL string `json:"html_url"`
	}
	if err := doJSON(ctx, p.Client, endpoint, map[string]string{"Authorization": "Bearer " + p.Token}, payload, &response); err != nil {
		return "", err
	}
	return response.HTMLURL, nil
}

func (p *GitLabProvider) Clone(context.Context, Repository, string) (string, error) {
	return "", errors.New("GitLabProvider only supports pull request API operations")
}

func (p *GitLabProvider) CreateBranch(context.Context, string, string) error {
	return errors.New("GitLabProvider only supports pull request API operations")
}

func (p *GitLabProvider) CommitFiles(context.Context, CommitRequest) error {
	return errors.New("GitLabProvider only supports pull request API operations")
}

func (p *GitLabProvider) OpenPullRequest(ctx context.Context, req PullRequestRequest) (string, error) {
	project := url.PathEscape(req.Repository.Owner + "/" + req.Repository.Name)
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/api/v4/projects/" + project + "/merge_requests"
	payload := map[string]string{"title": req.Title, "source_branch": req.HeadBranch, "target_branch": req.BaseBranch, "description": req.Body}
	var response struct {
		WebURL string `json:"web_url"`
	}
	if err := doJSON(ctx, p.Client, endpoint, map[string]string{"PRIVATE-TOKEN": p.Token}, payload, &response); err != nil {
		return "", err
	}
	return response.WebURL, nil
}

func doJSON(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("git provider API returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func envDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
