package skills

import (
	"testing"
)

func TestParseSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    *GitHubSource
		wantErr bool
	}{
		{
			name:   "shorthand with path",
			source: "trailofbits/skills/plugins/differential-review",
			want: &GitHubSource{
				Owner: "trailofbits",
				Repo:  "skills",
				Path:  "plugins/differential-review",
			},
		},
		{
			name:   "shorthand repo only",
			source: "anthropics/skills",
			want: &GitHubSource{
				Owner: "anthropics",
				Repo:  "skills",
				Path:  "",
			},
		},
		{
			name:   "shorthand with single path segment",
			source: "owner/repo/skill-name",
			want: &GitHubSource{
				Owner: "owner",
				Repo:  "repo",
				Path:  "skill-name",
			},
		},
		{
			name:   "full URL with tree/main",
			source: "https://github.com/anthropics/skills/tree/main/pdf-processing",
			want: &GitHubSource{
				Owner: "anthropics",
				Repo:  "skills",
				Path:  "pdf-processing",
			},
		},
		{
			name:   "full URL with tree/master and deep path",
			source: "https://github.com/trailofbits/skills/tree/master/plugins/security/review",
			want: &GitHubSource{
				Owner: "trailofbits",
				Repo:  "skills",
				Path:  "plugins/security/review",
			},
		},
		{
			name:   "full URL repo root",
			source: "https://github.com/owner/repo",
			want: &GitHubSource{
				Owner: "owner",
				Repo:  "repo",
				Path:  "",
			},
		},
		{
			name:   "full URL with blob",
			source: "https://github.com/owner/repo/blob/main/skills/test/SKILL.md",
			want: &GitHubSource{
				Owner: "owner",
				Repo:  "repo",
				Path:  "skills/test", // SKILL.md stripped
			},
		},
		{
			name:   "http URL (not https)",
			source: "http://github.com/owner/repo/tree/main/skill",
			want: &GitHubSource{
				Owner: "owner",
				Repo:  "repo",
				Path:  "skill",
			},
		},
		{
			name:   "shorthand with trailing SKILL.md",
			source: "owner/repo/path/SKILL.md",
			want: &GitHubSource{
				Owner: "owner",
				Repo:  "repo",
				Path:  "path",
			},
		},
		{
			name:    "empty source",
			source:  "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			source:  "   ",
			wantErr: true,
		},
		{
			name:    "single segment",
			source:  "just-one",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			source:  "https://gitlab.com/owner/repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Owner != tt.want.Owner {
				t.Errorf("Owner = %q, want %q", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.want.Repo)
			}
			if got.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.want.Path)
			}
		})
	}
}

func TestGitHubSource_String(t *testing.T) {
	tests := []struct {
		name   string
		source GitHubSource
		want   string
	}{
		{
			name:   "with path",
			source: GitHubSource{Owner: "owner", Repo: "repo", Path: "path/to/skill"},
			want:   "owner/repo/path/to/skill",
		},
		{
			name:   "without path",
			source: GitHubSource{Owner: "owner", Repo: "repo", Path: ""},
			want:   "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGitHubSource_APIURL(t *testing.T) {
	source := GitHubSource{Owner: "trailofbits", Repo: "skills", Path: "plugins/test"}
	want := "https://api.github.com/repos/trailofbits/skills"
	if got := source.APIURL(); got != want {
		t.Errorf("APIURL() = %q, want %q", got, want)
	}
}

func TestGitHubSource_RawContentURL(t *testing.T) {
	tests := []struct {
		name   string
		source GitHubSource
		commit string
		want   string
	}{
		{
			name:   "with path",
			source: GitHubSource{Owner: "owner", Repo: "repo", Path: "path/to/skill"},
			commit: "abc123",
			want:   "https://raw.githubusercontent.com/owner/repo/abc123/path/to/skill/SKILL.md",
		},
		{
			name:   "at root",
			source: GitHubSource{Owner: "owner", Repo: "repo", Path: ""},
			commit: "def456",
			want:   "https://raw.githubusercontent.com/owner/repo/def456/SKILL.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.RawContentURL(tt.commit); got != tt.want {
				t.Errorf("RawContentURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewGitHubClient(t *testing.T) {
	// Test with explicit token
	client := NewGitHubClient("my-token")
	if client.Token != "my-token" {
		t.Errorf("Token = %q, want %q", client.Token, "my-token")
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
	if client.HTTPClient.Timeout != DefaultHTTPTimeout {
		t.Errorf("Timeout = %v, want %v", client.HTTPClient.Timeout, DefaultHTTPTimeout)
	}

	// Test with empty token (uses env, but we don't set it in test)
	client2 := NewGitHubClient("")
	if client2.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
}
