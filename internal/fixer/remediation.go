package fixer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	RemediationStatusPROpened         = "pr_opened"
	RemediationStatusFailedValidation = "failed_validation"
)

type Repository struct {
	Provider   string `json:"provider"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	CloneURL   string `json:"clone_url"`
	BaseBranch string `json:"base_branch"`
}

type VulnerabilityReport struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	Endpoint    string `json:"endpoint"`
}

type RemediationRequest struct {
	Repository    Repository          `json:"repository"`
	Vulnerability VulnerabilityReport `json:"vulnerability"`
}

type RemediationResult struct {
	Status         string   `json:"status"`
	Branch         string   `json:"branch"`
	PullRequestURL string   `json:"pull_request_url,omitempty"`
	Logs           []string `json:"logs,omitempty"`
}

type CommitRequest struct {
	Repository Repository
	Branch     string
	Files      []string
	Message    string
}

type PullRequestRequest struct {
	Repository Repository
	HeadBranch string
	BaseBranch string
	Title      string
	Body       string
}

type MiniScanRequest struct {
	Repository    Repository
	Branch        string
	WorkspacePath string
	Vulnerability VulnerabilityReport
}

type MiniScanResult struct {
	VulnerabilityStillPresent bool
	Findings                  []string
}

type GitProvider interface {
	Clone(ctx context.Context, repo Repository, baseBranch string) (string, error)
	CreateBranch(ctx context.Context, workspacePath string, branch string) error
	CommitFiles(ctx context.Context, req CommitRequest) error
	OpenPullRequest(ctx context.Context, req PullRequestRequest) (string, error)
}

type Scanner interface {
	RunMiniScan(ctx context.Context, req MiniScanRequest) (MiniScanResult, error)
}

type RemediationActivity struct {
	Git     GitProvider
	Scanner Scanner
}

func BranchNameForVulnerability(vulnID string) string {
	return "aegis-fix-" + strings.TrimSpace(vulnID)
}

func (a RemediationActivity) RemediateVulnerability(ctx context.Context, req RemediationRequest) (RemediationResult, error) {
	if a.Git == nil {
		return RemediationResult{}, errors.New("git provider is required")
	}
	if a.Scanner == nil {
		return RemediationResult{}, errors.New("scanner is required")
	}
	if !isSQLInjection(req.Vulnerability.Type) {
		return RemediationResult{}, fmt.Errorf("unsupported vulnerability type %q", req.Vulnerability.Type)
	}

	baseBranch := strings.TrimSpace(req.Repository.BaseBranch)
	if baseBranch == "" {
		baseBranch = "main"
	}
	branch := BranchNameForVulnerability(req.Vulnerability.ID)
	logs := []string{"starting remediation", "branch=" + branch}

	workspacePath, err := a.Git.Clone(ctx, req.Repository, baseBranch)
	if err != nil {
		return RemediationResult{}, fmt.Errorf("clone repository: %w", err)
	}
	if err := a.Git.CreateBranch(ctx, workspacePath, branch); err != nil {
		return RemediationResult{}, fmt.Errorf("create branch: %w", err)
	}

	changedFiles, err := ApplySQLiPatch(workspacePath, req.Vulnerability)
	if err != nil {
		return RemediationResult{}, err
	}
	logs = append(logs, fmt.Sprintf("patched_files=%d", len(changedFiles)))

	scan, err := a.Scanner.RunMiniScan(ctx, MiniScanRequest{
		Repository:    req.Repository,
		Branch:        branch,
		WorkspacePath: workspacePath,
		Vulnerability: req.Vulnerability,
	})
	if err != nil {
		return RemediationResult{}, fmt.Errorf("mini-scan failed: %w", err)
	}
	if scan.VulnerabilityStillPresent {
		logs = append(logs, "mini-scan still detects vulnerability")
		return RemediationResult{Status: RemediationStatusFailedValidation, Branch: branch, Logs: logs}, nil
	}

	if err := a.Git.CommitFiles(ctx, CommitRequest{
		Repository: req.Repository,
		Branch:     branch,
		Files:      changedFiles,
		Message:    "[FIX] Remediate " + req.Vulnerability.ID,
	}); err != nil {
		return RemediationResult{}, fmt.Errorf("commit files: %w", err)
	}

	prURL, err := a.Git.OpenPullRequest(ctx, PullRequestRequest{
		Repository: req.Repository,
		HeadBranch: branch,
		BaseBranch: baseBranch,
		Title:      "[FIX] Remediate " + req.Vulnerability.ID,
		Body:       pullRequestBody(req.Vulnerability),
	})
	if err != nil {
		return RemediationResult{}, fmt.Errorf("open pull request: %w", err)
	}
	logs = append(logs, "pull_request="+prURL)

	return RemediationResult{Status: RemediationStatusPROpened, Branch: branch, PullRequestURL: prURL, Logs: logs}, nil
}

func ApplySQLiPatch(workspacePath string, report VulnerabilityReport) ([]string, error) {
	if !isSQLInjection(report.Type) {
		return nil, fmt.Errorf("unsupported vulnerability type %q", report.Type)
	}

	path := filepath.Clean(report.FilePath)
	if path == "." || strings.HasPrefix(path, "..") || filepath.IsAbs(path) {
		return nil, errors.New("vulnerability report must include a repository-relative file_path")
	}
	absPath := filepath.Join(workspacePath, path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read vulnerable file: %w", err)
	}

	patched := strings.ReplaceAll(string(content), "SELECT * FROM users WHERE email = '\" + email + \"'", "SELECT * FROM users WHERE email = ?")
	patched = strings.ReplaceAll(patched, "SELECT * FROM users WHERE email = '\" + q + \"'", "SELECT * FROM users WHERE email = ?")
	if patched == string(content) {
		return nil, errors.New("no supported SQLi pattern found")
	}
	if err := os.WriteFile(absPath, []byte(patched), 0o600); err != nil {
		return nil, fmt.Errorf("write patched file: %w", err)
	}
	return []string{absPath}, nil
}

func isSQLInjection(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(normalized, "sql injection") || strings.Contains(normalized, "sqli")
}

func pullRequestBody(v VulnerabilityReport) string {
	return fmt.Sprintf("Aegis generated this remediation for vulnerability `%s`.\n\nType: %s\n\nMini-scan validation passed before PR creation.", v.ID, v.Type)
}
