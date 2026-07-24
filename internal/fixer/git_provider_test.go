package fixer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewGitProviderFromEnvSelectsGitHub(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token")
	t.Setenv("GITLAB_TOKEN", "")

	provider, err := NewGitProviderFromEnv(http.DefaultClient)
	if err != nil {
		t.Fatalf("NewGitProviderFromEnv error: %v", err)
	}
	if _, ok := provider.(*GitHubProvider); !ok {
		t.Fatalf("provider type = %T, want *GitHubProvider", provider)
	}
}

func TestNewActivitySetUsesConfiguredProvider(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token")
	set, err := NewActivitySet(&fakeScanner{})
	if err != nil {
		t.Fatalf("NewActivitySet error: %v", err)
	}
	local, ok := set.Remediation.Git.(*LocalGitProvider)
	if !ok {
		t.Fatalf("git provider type = %T", set.Remediation.Git)
	}
	if _, ok := local.PRProvider.(*GitHubProvider); !ok {
		t.Fatalf("PR provider type = %T", local.PRProvider)
	}
}

func TestNewActivitySetReturnsProviderError(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITLAB_TOKEN", "")
	_, err := NewActivitySet(&fakeScanner{})
	if err == nil {
		t.Fatal("expected provider error")
	}
}

func TestGitHubProviderOpenPullRequestPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/repos/acme/demo/pulls" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"html_url":"https://github.example/acme/demo/pull/7"}`))
	}))
	defer server.Close()

	provider := &GitHubProvider{Token: "token", BaseURL: server.URL, Client: server.Client()}
	url, err := provider.OpenPullRequest(context.Background(), PullRequestRequest{
		Repository: Repository{Owner: "acme", Name: "demo"},
		HeadBranch: "aegis-fix-SQLI-123",
		BaseBranch: "main",
		Title:      "fix title",
		Body:       "body",
	})
	if err != nil {
		t.Fatalf("OpenPullRequest error: %v", err)
	}
	if url != "https://github.example/acme/demo/pull/7" {
		t.Fatalf("url = %q", url)
	}
	if payload["head"] != "aegis-fix-SQLI-123" || payload["base"] != "main" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestGitLabProviderOpenPullRequestPayload(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/api/v4/projects/acme%2Fdemo/merge_requests" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "token" {
			t.Fatalf("PRIVATE-TOKEN = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"web_url":"https://gitlab.example/acme/demo/-/merge_requests/7"}`))
	}))
	defer server.Close()

	provider := &GitLabProvider{Token: "token", BaseURL: server.URL, Client: server.Client()}
	url, err := provider.OpenPullRequest(context.Background(), PullRequestRequest{
		Repository: Repository{Owner: "acme", Name: "demo"},
		HeadBranch: "aegis-fix-SQLI-123",
		BaseBranch: "main",
		Title:      "fix title",
		Body:       "body",
	})
	if err != nil {
		t.Fatalf("OpenPullRequest error: %v", err)
	}
	if url != "https://gitlab.example/acme/demo/-/merge_requests/7" {
		t.Fatalf("url = %q", url)
	}
	if payload["source_branch"] != "aegis-fix-SQLI-123" || payload["target_branch"] != "main" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestNewGitProviderFromEnvRequiresToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITLAB_TOKEN", "")
	_, err := NewGitProviderFromEnv(http.DefaultClient)
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestNewGitProviderFromEnvSelectsGitLab(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITLAB_TOKEN", "token")
	t.Setenv("GITLAB_API_URL", "https://gitlab.internal")

	provider, err := NewGitProviderFromEnv(http.DefaultClient)
	if err != nil {
		t.Fatalf("NewGitProviderFromEnv error: %v", err)
	}
	gitlab, ok := provider.(*GitLabProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *GitLabProvider", provider)
	}
	if gitlab.BaseURL != "https://gitlab.internal" {
		t.Fatalf("BaseURL = %q", gitlab.BaseURL)
	}
}

func TestGitHubProviderOpenPullRequestRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	provider := &GitHubProvider{Token: "token", BaseURL: server.URL, Client: server.Client()}
	_, err := provider.OpenPullRequest(context.Background(), PullRequestRequest{Repository: Repository{Owner: "acme", Name: "demo"}})
	if err == nil {
		t.Fatal("expected GitHub HTTP error")
	}
}

func TestAPIOnlyProvidersRejectLocalGitOperations(t *testing.T) {
	github := &GitHubProvider{}
	gitlab := &GitLabProvider{}
	if _, err := github.Clone(context.Background(), Repository{}, "main"); err == nil {
		t.Fatal("expected github clone error")
	}
	if err := github.CreateBranch(context.Background(), "", "branch"); err == nil {
		t.Fatal("expected github create branch error")
	}
	if err := github.CommitFiles(context.Background(), CommitRequest{}); err == nil {
		t.Fatal("expected github commit error")
	}
	if _, err := gitlab.Clone(context.Background(), Repository{}, "main"); err == nil {
		t.Fatal("expected gitlab clone error")
	}
	if err := gitlab.CreateBranch(context.Background(), "", "branch"); err == nil {
		t.Fatal("expected gitlab create branch error")
	}
	if err := gitlab.CommitFiles(context.Background(), CommitRequest{}); err == nil {
		t.Fatal("expected gitlab commit error")
	}
}

func TestLocalGitProviderCreatesBranchCommitsAndDelegatesPR(t *testing.T) {
	ctx := context.Background()
	remote := initRemoteRepo(t)
	provider := &LocalGitProvider{PRProvider: &fakeGitProvider{pullRequestURL: "https://git.example/pr/9"}}

	workspace, err := provider.Clone(ctx, Repository{CloneURL: remote}, "main")
	if err != nil {
		t.Fatalf("Clone error: %v", err)
	}
	runGit(t, workspace, "config", "user.email", "aegis@example.local")
	runGit(t, workspace, "config", "user.name", "Aegis Test")
	if err := provider.CreateBranch(ctx, workspace, "aegis-fix-SQLI-123"); err != nil {
		t.Fatalf("CreateBranch error: %v", err)
	}
	file := filepath.Join(workspace, "handler.go")
	if err := os.WriteFile(file, []byte("patched"), 0o600); err != nil {
		t.Fatalf("write patch: %v", err)
	}
	if err := provider.CommitFiles(ctx, CommitRequest{Branch: "aegis-fix-SQLI-123", Files: []string{file}, Message: "[FIX] patch"}); err != nil {
		t.Fatalf("CommitFiles error: %v", err)
	}
	url, err := provider.OpenPullRequest(ctx, PullRequestRequest{HeadBranch: "aegis-fix-SQLI-123"})
	if err != nil {
		t.Fatalf("OpenPullRequest error: %v", err)
	}
	if url != "https://git.example/pr/9" {
		t.Fatalf("url = %q", url)
	}
}

func initRemoteRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	work := filepath.Join(root, "work")
	remote := filepath.Join(root, "remote.git")
	runGit(t, root, "init", "--bare", remote)
	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatalf("mkdir work: %v", err)
	}
	runGit(t, work, "init", "-b", "main")
	runGit(t, work, "config", "user.email", "aegis@example.local")
	runGit(t, work, "config", "user.name", "Aegis Test")
	if err := os.WriteFile(filepath.Join(work, "handler.go"), []byte("original"), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runGit(t, work, "add", "handler.go")
	runGit(t, work, "commit", "-m", "initial")
	runGit(t, work, "remote", "add", "origin", remote)
	runGit(t, work, "push", "-u", "origin", "main")
	return remote
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, out)
	}
}
