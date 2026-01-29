package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSkillMDBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantName    string
		wantDesc    string
		wantTags    []string
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "full skill file",
			input: `---
name: differential-review
description: >
  Security-focused differential review of code changes.
tags: ["security", "code-review"]
---

# Differential Review

This is the skill body.`,
			wantName: "differential-review",
			wantDesc: "Security-focused differential review of code changes.\n",
			wantTags: []string{"security", "code-review"},
			wantBody: "# Differential Review\n\nThis is the skill body.",
			wantErr:  false,
		},
		{
			name: "minimal frontmatter",
			input: `---
name: simple
description: A simple skill
---

Content here.`,
			wantName: "simple",
			wantDesc: "A simple skill",
			wantTags: nil,
			wantBody: "Content here.",
			wantErr:  false,
		},
		{
			name:     "no frontmatter",
			input:    "# Just Markdown\n\nNo frontmatter here.",
			wantName: "",
			wantDesc: "",
			wantBody: "# Just Markdown\n\nNo frontmatter here.",
			wantErr:  false,
		},
		{
			name:     "empty file",
			input:    "",
			wantName: "",
			wantDesc: "",
			wantBody: "",
			wantErr:  false,
		},
		{
			name: "unclosed frontmatter",
			input: `---
name: broken
description: Missing closing delimiter`,
			wantErr:     true,
			errContains: "unclosed frontmatter",
		},
		{
			name: "frontmatter with metadata",
			input: `---
name: advanced
description: An advanced skill
license: MIT
metadata:
  author: test-author
  version: "1.0"
---

Body content.`,
			wantName: "advanced",
			wantDesc: "An advanced skill",
			wantBody: "Body content.",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill, err := ParseSkillMDBytes([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if skill.Frontmatter.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", skill.Frontmatter.Name, tt.wantName)
			}
			if skill.Frontmatter.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", skill.Frontmatter.Description, tt.wantDesc)
			}
			if skill.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", skill.Body, tt.wantBody)
			}
			if tt.wantTags != nil {
				if len(skill.Frontmatter.Tags) != len(tt.wantTags) {
					t.Errorf("len(Tags) = %d, want %d", len(skill.Frontmatter.Tags), len(tt.wantTags))
				} else {
					for i, tag := range tt.wantTags {
						if skill.Frontmatter.Tags[i] != tag {
							t.Errorf("Tags[%d] = %q, want %q", i, skill.Frontmatter.Tags[i], tag)
						}
					}
				}
			}
		})
	}
}

func TestParseSkillMDFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")

	content := `---
name: file-test
description: Testing file parsing
tags: ["test"]
---

# File Test Skill

This tests file parsing.`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	skill, err := ParseSkillMDFile(path)
	if err != nil {
		t.Fatalf("ParseSkillMDFile() error = %v", err)
	}

	if skill.Frontmatter.Name != "file-test" {
		t.Errorf("Name = %q, want %q", skill.Frontmatter.Name, "file-test")
	}
	if skill.Frontmatter.Description != "Testing file parsing" {
		t.Errorf("Description = %q, want %q", skill.Frontmatter.Description, "Testing file parsing")
	}
	if skill.Raw != content {
		t.Errorf("Raw content not preserved")
	}
}

func TestParseSkillMDFileNotFound(t *testing.T) {
	_, err := ParseSkillMDFile("/nonexistent/SKILL.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseSkillMD(t *testing.T) {
	content := `---
name: reader-test
description: Testing reader parsing
---

Body.`

	skill, err := ParseSkillMD(strings.NewReader(content))
	if err != nil {
		t.Fatalf("ParseSkillMD() error = %v", err)
	}

	if skill.Frontmatter.Name != "reader-test" {
		t.Errorf("Name = %q, want %q", skill.Frontmatter.Name, "reader-test")
	}
}

func TestFrontmatterValidate(t *testing.T) {
	tests := []struct {
		name        string
		fm          Frontmatter
		wantErr     bool
		errContains string
	}{
		{
			name: "valid",
			fm: Frontmatter{
				Name:        "test",
				Description: "A test skill",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			fm: Frontmatter{
				Description: "A test skill",
			},
			wantErr:     true,
			errContains: "name",
		},
		{
			name: "missing description",
			fm: Frontmatter{
				Name: "test",
			},
			wantErr:     true,
			errContains: "description",
		},
		{
			name:        "empty",
			fm:          Frontmatter{},
			wantErr:     true,
			errContains: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fm.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSkillMDValidate(t *testing.T) {
	skill := &SkillMD{
		Frontmatter: Frontmatter{
			Name:        "test",
			Description: "A test skill",
		},
		Body: "# Test",
	}

	if err := skill.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	// Invalid
	skill.Frontmatter.Name = ""
	if err := skill.Validate(); err == nil {
		t.Error("Validate() expected error for missing name")
	}
}

func TestExtractFrontmatterEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFM   string
		wantBody string
		wantErr  bool
	}{
		{
			name:     "frontmatter only",
			input:    "---\nname: test\n---",
			wantFM:   "name: test",
			wantBody: "",
			wantErr:  false,
		},
		{
			name:     "body only no marker",
			input:    "Just some text",
			wantFM:   "",
			wantBody: "Just some text",
			wantErr:  false,
		},
		{
			name:     "multiple blank lines after frontmatter",
			input:    "---\nname: test\n---\n\n\n\nBody",
			wantFM:   "name: test",
			wantBody: "Body",
			wantErr:  false,
		},
		{
			name:     "frontmatter with empty lines",
			input:    "---\nname: test\n\ndescription: desc\n---\n\nBody",
			wantFM:   "name: test\n\ndescription: desc",
			wantBody: "Body",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := extractFrontmatter([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(fm) != tt.wantFM {
				t.Errorf("frontmatter = %q, want %q", string(fm), tt.wantFM)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestFrontmatterMetadata(t *testing.T) {
	content := `---
name: metadata-test
description: Testing metadata parsing
license: Apache-2.0
metadata:
  author: anthropic
  version: "2.0"
  repository: https://github.com/anthropics/skills
---

Body.`

	skill, err := ParseSkillMDBytes([]byte(content))
	if err != nil {
		t.Fatalf("ParseSkillMDBytes() error = %v", err)
	}

	if skill.Frontmatter.License != "Apache-2.0" {
		t.Errorf("License = %q, want %q", skill.Frontmatter.License, "Apache-2.0")
	}
	if skill.Frontmatter.Metadata.Author != "anthropic" {
		t.Errorf("Metadata.Author = %q, want %q", skill.Frontmatter.Metadata.Author, "anthropic")
	}
	if skill.Frontmatter.Metadata.Version != "2.0" {
		t.Errorf("Metadata.Version = %q, want %q", skill.Frontmatter.Metadata.Version, "2.0")
	}
	if skill.Frontmatter.Metadata.Repository != "https://github.com/anthropics/skills" {
		t.Errorf("Metadata.Repository = %q, want %q", skill.Frontmatter.Metadata.Repository, "https://github.com/anthropics/skills")
	}
}
