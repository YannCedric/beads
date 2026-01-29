// Package skills provides types and parsing for agent skill manifests.
//
// It handles both:
//   - skills.yaml: The project manifest listing installed skills (team-shared or local)
//   - SKILL.md: Individual skill files with YAML frontmatter
//
// See docs/design/skills.md for the full specification.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// File names for skill manifests
const (
	TeamFileName  = "skills.yaml"       // Git-committed, team-shared
	LocalFileName = "skills.local.yaml" // Git-ignored, personal
)

// SkillsFile represents the contents of skills.yaml or skills.local.yaml.
//
// Example:
//
//	version: 1
//	skills:
//	  differential-review:
//	    source: trailofbits/skills/plugins/differential-review
//	    commit: a1b2c3d4e5f6
//	    description: Security-focused differential review...
//	    tags: ["security", "code-review"]
type SkillsFile struct {
	Version int              `yaml:"version"`
	Skills  map[string]Skill `yaml:"skills,omitempty"`
}

// Skill represents a single installed skill entry in skills.yaml.
type Skill struct {
	// Source is the original install source (GitHub shorthand or full URL).
	// Examples: "trailofbits/skills/plugins/differential-review"
	//           "https://github.com/anthropics/skills/tree/main/pdf-processing"
	Source string `yaml:"source"`

	// Commit is the pinned commit hash for reproducibility.
	Commit string `yaml:"commit"`

	// Description is the skill's description from its SKILL.md frontmatter.
	// Used for progressive disclosure - agent sees this at startup (~100 tokens).
	Description string `yaml:"description"`

	// Tags are optional categorization labels from the skill.
	Tags []string `yaml:"tags,omitempty,flow"`

	// Size is the SKILL.md file size in bytes.
	Size int64 `yaml:"size,omitempty"`

	// InstalledAt is when the skill was installed.
	InstalledAt time.Time `yaml:"installed_at,omitempty"`

	// InstalledBy is the user who installed the skill (email or username).
	InstalledBy string `yaml:"installed_by,omitempty"`
}

// Load reads a skills file from the specified path.
// Returns nil, nil if the file does not exist.
func Load(path string) (*SkillsFile, error) {
	data, err := os.ReadFile(path) // #nosec G304 - controlled path from caller
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading skills file: %w", err)
	}

	var sf SkillsFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parsing skills file: %w", err)
	}

	// Initialize map if nil (empty file or version-only)
	if sf.Skills == nil {
		sf.Skills = make(map[string]Skill)
	}

	return &sf, nil
}

// LoadTeam loads the team skills file from the .beads directory.
// Returns an empty SkillsFile if the file doesn't exist.
func LoadTeam(beadsDir string) (*SkillsFile, error) {
	sf, err := Load(filepath.Join(beadsDir, TeamFileName))
	if err != nil {
		return nil, err
	}
	if sf == nil {
		return &SkillsFile{
			Version: 1,
			Skills:  make(map[string]Skill),
		}, nil
	}
	return sf, nil
}

// LoadLocal loads the local (personal) skills file from the .beads directory.
// Returns an empty SkillsFile if the file doesn't exist.
func LoadLocal(beadsDir string) (*SkillsFile, error) {
	sf, err := Load(filepath.Join(beadsDir, LocalFileName))
	if err != nil {
		return nil, err
	}
	if sf == nil {
		return &SkillsFile{
			Version: 1,
			Skills:  make(map[string]Skill),
		}, nil
	}
	return sf, nil
}

// LoadAll loads both team and local skills, merging them into a single map.
// Local skills take precedence over team skills with the same name.
func LoadAll(beadsDir string) (map[string]Skill, error) {
	team, err := LoadTeam(beadsDir)
	if err != nil {
		return nil, fmt.Errorf("loading team skills: %w", err)
	}

	local, err := LoadLocal(beadsDir)
	if err != nil {
		return nil, fmt.Errorf("loading local skills: %w", err)
	}

	// Merge: start with team, overlay local
	merged := make(map[string]Skill, len(team.Skills)+len(local.Skills))
	for name, skill := range team.Skills {
		merged[name] = skill
	}
	for name, skill := range local.Skills {
		merged[name] = skill
	}

	return merged, nil
}

// Save writes the skills file to the specified path.
func (sf *SkillsFile) Save(path string) error {
	// Ensure version is set
	if sf.Version == 0 {
		sf.Version = 1
	}

	data, err := yaml.Marshal(sf)
	if err != nil {
		return fmt.Errorf("marshaling skills file: %w", err)
	}

	// #nosec G306 - skills.yaml is meant to be git-committed and team-readable
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing skills file: %w", err)
	}

	return nil
}

// SaveTeam saves the skills file to the team location in .beads.
func (sf *SkillsFile) SaveTeam(beadsDir string) error {
	return sf.Save(filepath.Join(beadsDir, TeamFileName))
}

// SaveLocal saves the skills file to the local location in .beads.
func (sf *SkillsFile) SaveLocal(beadsDir string) error {
	return sf.Save(filepath.Join(beadsDir, LocalFileName))
}

// TeamPath returns the path to the team skills file.
func TeamPath(beadsDir string) string {
	return filepath.Join(beadsDir, TeamFileName)
}

// LocalPath returns the path to the local skills file.
func LocalPath(beadsDir string) string {
	return filepath.Join(beadsDir, LocalFileName)
}

// Add adds or updates a skill in the skills file.
func (sf *SkillsFile) Add(name string, skill Skill) {
	if sf.Skills == nil {
		sf.Skills = make(map[string]Skill)
	}
	sf.Skills[name] = skill
}

// Remove removes a skill from the skills file.
// Returns true if the skill was found and removed.
func (sf *SkillsFile) Remove(name string) bool {
	if sf.Skills == nil {
		return false
	}
	if _, exists := sf.Skills[name]; !exists {
		return false
	}
	delete(sf.Skills, name)
	return true
}

// Get returns a skill by name and whether it exists.
func (sf *SkillsFile) Get(name string) (Skill, bool) {
	if sf.Skills == nil {
		return Skill{}, false
	}
	skill, exists := sf.Skills[name]
	return skill, exists
}

// Names returns the names of all skills in alphabetical order.
func (sf *SkillsFile) Names() []string {
	if sf.Skills == nil {
		return nil
	}
	names := make([]string, 0, len(sf.Skills))
	for name := range sf.Skills {
		names = append(names, name)
	}
	// Sort for consistent output
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}
