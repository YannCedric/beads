package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// GitHubSource represents a parsed GitHub skill source.
type GitHubSource struct {
	Owner string // Repository owner (user or org)
	Repo  string // Repository name
	Path  string // Path within the repository to the skill directory
}

// DefaultHTTPTimeout is the timeout for GitHub API requests.
const DefaultHTTPTimeout = 30 * time.Second

// FetchResult contains the result of fetching a skill from GitHub.
type FetchResult struct {
	Content  []byte // SKILL.md content
	Commit   string // Commit SHA
	SkillMD  *SkillMD
}

// gitHubURLRegex matches GitHub URLs like:
// - https://github.com/owner/repo/tree/main/path/to/skill
// - https://github.com/owner/repo/blob/main/path/to/SKILL.md
var gitHubURLRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)(?:/(?:tree|blob)/[^/]+)?(?:/(.*))?$`)

// shorthandRegex matches GitHub shorthand like:
// - owner/repo/path/to/skill
// - owner/repo (skill at root)
var shorthandRegex = regexp.MustCompile(`^([^/]+)/([^/]+)(?:/(.*))?$`)

// ParseSource parses a GitHub source string into its components.
// Accepts:
//   - GitHub shorthand: "owner/repo/path/to/skill"
//   - Full GitHub URL: "https://github.com/owner/repo/tree/main/path/to/skill"
func ParseSource(source string) (*GitHubSource, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("empty source")
	}

	// Try full URL first
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		matches := gitHubURLRegex.FindStringSubmatch(source)
		if matches == nil {
			return nil, fmt.Errorf("invalid GitHub URL: %s", source)
		}
		return &GitHubSource{
			Owner: matches[1],
			Repo:  matches[2],
			Path:  strings.TrimSuffix(strings.TrimSuffix(matches[3], "/SKILL.md"), "/"),
		}, nil
	}

	// Try shorthand
	matches := shorthandRegex.FindStringSubmatch(source)
	if matches == nil {
		return nil, fmt.Errorf("invalid source format: %s (expected owner/repo/path or GitHub URL)", source)
	}

	return &GitHubSource{
		Owner: matches[1],
		Repo:  matches[2],
		Path:  strings.TrimSuffix(strings.TrimSuffix(matches[3], "/SKILL.md"), "/"),
	}, nil
}

// String returns the canonical shorthand form of the source.
func (s *GitHubSource) String() string {
	if s.Path == "" {
		return fmt.Sprintf("%s/%s", s.Owner, s.Repo)
	}
	return fmt.Sprintf("%s/%s/%s", s.Owner, s.Repo, s.Path)
}

// APIURL returns the GitHub API URL for the repository.
func (s *GitHubSource) APIURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s", s.Owner, s.Repo)
}

// RawContentURL returns the raw.githubusercontent.com URL for SKILL.md.
func (s *GitHubSource) RawContentURL(commitSHA string) string {
	if s.Path == "" {
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/SKILL.md", s.Owner, s.Repo, commitSHA)
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/SKILL.md", s.Owner, s.Repo, commitSHA, s.Path)
}

// GitHubClient handles GitHub API requests.
type GitHubClient struct {
	HTTPClient *http.Client
	Token      string // Optional GitHub token for auth
}

// NewGitHubClient creates a new GitHub client.
// If token is empty, it checks the GITHUB_TOKEN environment variable.
func NewGitHubClient(token string) *GitHubClient {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	return &GitHubClient{
		HTTPClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
		Token: token,
	}
}

// doRequest executes an HTTP request with proper headers.
func (c *GitHubClient) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "beads-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}

	return c.HTTPClient.Do(req)
}

// GetDefaultBranch fetches the default branch name for a repository.
func (c *GitHubClient) GetDefaultBranch(ctx context.Context, source *GitHubSource) (string, error) {
	resp, err := c.doRequest(ctx, "GET", source.APIURL())
	if err != nil {
		return "", fmt.Errorf("fetching repository info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("repository not found: %s/%s", source.Owner, source.Repo)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if c.Token == "" {
			return "", fmt.Errorf("access denied: set GITHUB_TOKEN for private repositories")
		}
		return "", fmt.Errorf("access denied: check your GITHUB_TOKEN permissions")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", fmt.Errorf("parsing repository info: %w", err)
	}

	return repoInfo.DefaultBranch, nil
}

// GetLatestCommit fetches the latest commit SHA for a branch.
func (c *GitHubClient) GetLatestCommit(ctx context.Context, source *GitHubSource, branch string) (string, error) {
	url := fmt.Sprintf("%s/commits/%s", source.APIURL(), branch)
	resp, err := c.doRequest(ctx, "GET", url)
	if err != nil {
		return "", fmt.Errorf("fetching commit info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error fetching commit: %s", resp.Status)
	}

	var commitInfo struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commitInfo); err != nil {
		return "", fmt.Errorf("parsing commit info: %w", err)
	}

	return commitInfo.SHA, nil
}

// DownloadSkillMD downloads the SKILL.md file content.
func (c *GitHubClient) DownloadSkillMD(ctx context.Context, source *GitHubSource, commitSHA string) ([]byte, error) {
	url := source.RawContentURL(commitSHA)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "beads-cli")
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading SKILL.md: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		if source.Path == "" {
			return nil, fmt.Errorf("SKILL.md not found at repository root")
		}
		return nil, fmt.Errorf("SKILL.md not found at path: %s", source.Path)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download SKILL.md: %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md content: %w", err)
	}

	return content, nil
}

// FetchSkill fetches a skill from GitHub, returning content and commit hash.
func (c *GitHubClient) FetchSkill(ctx context.Context, source *GitHubSource) (*FetchResult, error) {
	// Get default branch
	branch, err := c.GetDefaultBranch(ctx, source)
	if err != nil {
		return nil, err
	}

	// Get latest commit
	commit, err := c.GetLatestCommit(ctx, source, branch)
	if err != nil {
		return nil, err
	}

	// Download SKILL.md
	content, err := c.DownloadSkillMD(ctx, source, commit)
	if err != nil {
		return nil, err
	}

	// Parse SKILL.md
	skillMD, err := ParseSkillMDBytes(content)
	if err != nil {
		return nil, fmt.Errorf("parsing SKILL.md: %w", err)
	}

	// Validate
	if err := skillMD.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SKILL.md: %w", err)
	}

	return &FetchResult{
		Content:  content,
		Commit:   commit,
		SkillMD:  skillMD,
	}, nil
}

// ResolveAndFetch parses a source string and fetches the skill.
func ResolveAndFetch(ctx context.Context, source string) (*GitHubSource, *FetchResult, error) {
	parsed, err := ParseSource(source)
	if err != nil {
		return nil, nil, err
	}

	client := NewGitHubClient("")
	result, err := client.FetchSkill(ctx, parsed)
	if err != nil {
		return parsed, nil, err
	}

	return parsed, result, nil
}
