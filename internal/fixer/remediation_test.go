package fixer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeGitProvider struct {
	cloneDir       string
	branches       []string
	commits        []CommitRequest
	pullRequests   []PullRequestRequest
	pullRequestURL string
}

func (f *fakeGitProvider) Clone(_ context.Context, _ Repository, branch string) (string, error) {
	f.branches = append(f.branches, branch)
	return f.cloneDir, nil
}

func (f *fakeGitProvider) CreateBranch(_ context.Context, _ string, branch string) error {
	f.branches = append(f.branches, branch)
	return nil
}

func (f *fakeGitProvider) CommitFiles(_ context.Context, req CommitRequest) error {
	f.commits = append(f.commits, req)
	return nil
}

func (f *fakeGitProvider) OpenPullRequest(_ context.Context, req PullRequestRequest) (string, error) {
	f.pullRequests = append(f.pullRequests, req)
	if f.pullRequestURL == "" {
		return "https://git.example/pull/1", nil
	}
	return f.pullRequestURL, nil
}

type fakeScanner struct {
	result MiniScanResult
	calls  []MiniScanRequest
}

func (f *fakeScanner) RunMiniScan(_ context.Context, req MiniScanRequest) (MiniScanResult, error) {
	f.calls = append(f.calls, req)
	return f.result, nil
}

func writeVulnerableGoFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "handler.go")
	content := []byte(`package demo

func query(email string) string {
	return "SELECT * FROM users WHERE email = '" + email + "'"
}
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write vulnerable file: %v", err)
	}
	return path
}

func TestBranchNameForVulnerability(t *testing.T) {
	got := BranchNameForVulnerability("SQLI-123")
	if got != "aegis-fix-SQLI-123" {
		t.Fatalf("branch name = %q", got)
	}
}

func TestApplySQLiPatchRewritesStringConcatenation(t *testing.T) {
	dir := t.TempDir()
	path := writeVulnerableGoFile(t, dir)

	changed, err := ApplySQLiPatch(dir, VulnerabilityReport{ID: "SQLI-123", Type: "SQL Injection", FilePath: "handler.go"})
	if err != nil {
		t.Fatalf("ApplySQLiPatch error: %v", err)
	}
	if len(changed) != 1 || changed[0] != path {
		t.Fatalf("changed files = %#v, want [%s]", changed, path)
	}

	patched, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read patched file: %v", err)
	}
	if string(patched) != `package demo

func query(email string) string {
	return "SELECT * FROM users WHERE email = ?"
}
` {
		t.Fatalf("unexpected patch:\n%s", patched)
	}
}

func TestRemediationCreatesPullRequestAfterCleanMiniScan(t *testing.T) {
	dir := t.TempDir()
	writeVulnerableGoFile(t, dir)
	git := &fakeGitProvider{cloneDir: dir, pullRequestURL: "https://git.example/pr/42"}
	scanner := &fakeScanner{result: MiniScanResult{VulnerabilityStillPresent: false}}
	activity := RemediationActivity{Git: git, Scanner: scanner}

	result, err := activity.RemediateVulnerability(context.Background(), RemediationRequest{
		Repository:    Repository{Provider: "github", Owner: "acme", Name: "demo", CloneURL: "https://example/demo.git", BaseBranch: "main"},
		Vulnerability: VulnerabilityReport{ID: "SQLI-123", Type: "SQL Injection", FilePath: "handler.go", Title: "SQLi"},
	})
	if err != nil {
		t.Fatalf("RemediateVulnerability error: %v", err)
	}
	if result.Status != RemediationStatusPROpened || result.PullRequestURL != "https://git.example/pr/42" {
		t.Fatalf("result = %#v", result)
	}
	if len(scanner.calls) != 1 {
		t.Fatalf("mini-scan calls = %d, want 1", len(scanner.calls))
	}
	if len(git.pullRequests) != 1 {
		t.Fatalf("pull requests = %d, want 1", len(git.pullRequests))
	}
	if git.pullRequests[0].HeadBranch != "aegis-fix-SQLI-123" {
		t.Fatalf("head branch = %q", git.pullRequests[0].HeadBranch)
	}
}

func TestRemediationDoesNotCreatePullRequestWhenMiniScanStillFindsVulnerability(t *testing.T) {
	dir := t.TempDir()
	writeVulnerableGoFile(t, dir)
	git := &fakeGitProvider{cloneDir: dir}
	scanner := &fakeScanner{result: MiniScanResult{VulnerabilityStillPresent: true, Findings: []string{"SQLI-123"}}}
	activity := RemediationActivity{Git: git, Scanner: scanner}

	result, err := activity.RemediateVulnerability(context.Background(), RemediationRequest{
		Repository:    Repository{Provider: "github", Owner: "acme", Name: "demo", CloneURL: "https://example/demo.git", BaseBranch: "main"},
		Vulnerability: VulnerabilityReport{ID: "SQLI-123", Type: "SQL Injection", FilePath: "handler.go"},
	})
	if err != nil {
		t.Fatalf("RemediateVulnerability error: %v", err)
	}
	if result.Status != RemediationStatusFailedValidation {
		t.Fatalf("status = %q", result.Status)
	}
	if len(git.pullRequests) != 0 {
		t.Fatalf("pull requests = %d, want 0", len(git.pullRequests))
	}
}

func TestRemediationRejectsUnsupportedVulnerability(t *testing.T) {
	activity := RemediationActivity{Git: &fakeGitProvider{}, Scanner: &fakeScanner{}}
	_, err := activity.RemediateVulnerability(context.Background(), RemediationRequest{
		Repository:    Repository{Provider: "github", Owner: "acme", Name: "demo", CloneURL: "https://example/demo.git"},
		Vulnerability: VulnerabilityReport{ID: "XSS-1", Type: "XSS"},
	})
	if err == nil {
		t.Fatal("expected unsupported vulnerability error")
	}
}

func TestRemediationRequiresGitProviderAndScanner(t *testing.T) {
	_, err := (RemediationActivity{}).RemediateVulnerability(context.Background(), RemediationRequest{})
	if err == nil {
		t.Fatal("expected git provider error")
	}
	_, err = (RemediationActivity{Git: &fakeGitProvider{}}).RemediateVulnerability(context.Background(), RemediationRequest{})
	if err == nil {
		t.Fatal("expected scanner error")
	}
}

func TestApplySQLiPatchRejectsUnsafePath(t *testing.T) {
	_, err := ApplySQLiPatch(t.TempDir(), VulnerabilityReport{ID: "SQLI-123", Type: "SQLi", FilePath: "../handler.go"})
	if err == nil {
		t.Fatal("expected unsafe path error")
	}
}

func TestApplySQLiPatchRejectsMissingPattern(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "handler.go"), []byte("package demo\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := ApplySQLiPatch(dir, VulnerabilityReport{ID: "SQLI-123", Type: "SQLi", FilePath: "handler.go"})
	if err == nil {
		t.Fatal("expected missing pattern error")
	}
}
