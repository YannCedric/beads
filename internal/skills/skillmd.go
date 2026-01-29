package skills

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillMD represents a parsed SKILL.md file with YAML frontmatter.
//
// Example SKILL.md:
//
//	---
//	name: differential-review
//	description: >
//	  Security-focused differential review of code changes with git
//	  history analysis and blast radius estimation.
//	tags: ["security", "code-review"]
//	---
//
//	# Differential Review
//	...instructions...
type SkillMD struct {
	// Frontmatter contains the parsed YAML frontmatter.
	Frontmatter Frontmatter

	// Body contains the markdown content after the frontmatter.
	Body string

	// Raw contains the entire original file content.
	Raw string
}

// Frontmatter represents the YAML frontmatter in a SKILL.md file.
// Follows the Agent Skills specification: https://agentskills.io/specification
type Frontmatter struct {
	// Name is the skill identifier (required).
	Name string `yaml:"name"`

	// Description explains what the skill does and when to use it (required).
	// Should include trigger phrases like "Use when..." for agent activation.
	Description string `yaml:"description"`

	// License is the skill's license (optional, e.g., "MIT", "Apache-2.0").
	License string `yaml:"license,omitempty"`

	// Tags are categorization labels (optional).
	Tags []string `yaml:"tags,omitempty,flow"`

	// Metadata contains additional optional fields.
	Metadata FrontmatterMetadata `yaml:"metadata,omitempty"`
}

// FrontmatterMetadata contains optional metadata fields in SKILL.md frontmatter.
type FrontmatterMetadata struct {
	// Author is the skill author (person or organization).
	Author string `yaml:"author,omitempty"`

	// Version is the skill version string.
	Version string `yaml:"version,omitempty"`

	// Repository is the source repository URL.
	Repository string `yaml:"repository,omitempty"`
}

// ParseSkillMD parses a SKILL.md file from a reader.
// It extracts YAML frontmatter (between --- delimiters) and the markdown body.
func ParseSkillMD(r io.Reader) (*SkillMD, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading skill file: %w", err)
	}

	return ParseSkillMDBytes(data)
}

// ParseSkillMDBytes parses SKILL.md content from bytes.
func ParseSkillMDBytes(data []byte) (*SkillMD, error) {
	raw := string(data)

	frontmatter, body, err := extractFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var fm Frontmatter
	if len(frontmatter) > 0 {
		if err := yaml.Unmarshal(frontmatter, &fm); err != nil {
			return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
		}
	}

	return &SkillMD{
		Frontmatter: fm,
		Body:        body,
		Raw:         raw,
	}, nil
}

// ParseSkillMDFile parses a SKILL.md file from a file path.
func ParseSkillMDFile(path string) (*SkillMD, error) {
	data, err := os.ReadFile(path) // #nosec G304 - controlled path from caller
	if err != nil {
		return nil, fmt.Errorf("reading skill file: %w", err)
	}

	skill, err := ParseSkillMDBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return skill, nil
}

// extractFrontmatter extracts YAML frontmatter and body from markdown content.
// Frontmatter is delimited by --- at the start of the file.
//
// Returns:
//   - frontmatter: The YAML content between --- delimiters (empty if none)
//   - body: The markdown content after the frontmatter
//   - error: Any parsing error
func extractFrontmatter(data []byte) (frontmatter []byte, body string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// Check for opening ---
	if !scanner.Scan() {
		// Empty file
		return nil, "", nil
	}

	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine != "---" {
		// No frontmatter, entire content is body
		return nil, string(data), nil
	}

	// Collect frontmatter lines until closing ---
	var fmLines []string
	foundClosing := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClosing = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("scanning file: %w", err)
	}

	if !foundClosing {
		return nil, "", fmt.Errorf("unclosed frontmatter: missing closing ---")
	}

	// Join frontmatter lines
	frontmatter = []byte(strings.Join(fmLines, "\n"))

	// Collect remaining lines as body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("scanning file: %w", err)
	}

	// Join body lines, trimming leading empty lines
	body = strings.Join(bodyLines, "\n")
	body = strings.TrimLeft(body, "\n")

	return frontmatter, body, nil
}

// Validate checks that the frontmatter has required fields.
func (fm *Frontmatter) Validate() error {
	if fm.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if fm.Description == "" {
		return fmt.Errorf("missing required field: description")
	}
	return nil
}

// Validate checks that the SKILL.md has valid frontmatter.
func (s *SkillMD) Validate() error {
	return s.Frontmatter.Validate()
}
