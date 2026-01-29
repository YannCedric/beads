package skills

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadNonExistent(t *testing.T) {
	sf, err := Load("/nonexistent/path/skills.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if sf != nil {
		t.Errorf("Load() = %v, want nil for nonexistent file", sf)
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills.yaml")

	// Create a skills file
	sf := &SkillsFile{
		Version: 1,
		Skills: map[string]Skill{
			"test-skill": {
				Source:      "owner/repo/skills/test-skill",
				Commit:      "abc123",
				Description: "A test skill for testing",
				Tags:        []string{"test", "example"},
				Size:        1024,
				InstalledAt: time.Date(2026, 1, 28, 10, 0, 0, 0, time.UTC),
				InstalledBy: "user@example.com",
			},
		},
	}

	// Save
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Save() did not create file: %v", err)
	}

	// Load
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("Load() returned nil")
	}

	// Verify contents
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}
	if len(loaded.Skills) != 1 {
		t.Fatalf("len(Skills) = %d, want 1", len(loaded.Skills))
	}

	skill, ok := loaded.Skills["test-skill"]
	if !ok {
		t.Fatal("Skills[\"test-skill\"] not found")
	}
	if skill.Source != "owner/repo/skills/test-skill" {
		t.Errorf("Source = %q, want %q", skill.Source, "owner/repo/skills/test-skill")
	}
	if skill.Commit != "abc123" {
		t.Errorf("Commit = %q, want %q", skill.Commit, "abc123")
	}
	if skill.Description != "A test skill for testing" {
		t.Errorf("Description = %q, want %q", skill.Description, "A test skill for testing")
	}
	if len(skill.Tags) != 2 || skill.Tags[0] != "test" || skill.Tags[1] != "example" {
		t.Errorf("Tags = %v, want [test example]", skill.Tags)
	}
}

func TestLoadTeamAndLocal(t *testing.T) {
	dir := t.TempDir()

	// Create team skills file
	team := &SkillsFile{
		Version: 1,
		Skills: map[string]Skill{
			"team-skill": {
				Source:      "org/skills/team-skill",
				Commit:      "team123",
				Description: "A team skill",
			},
		},
	}
	if err := team.SaveTeam(dir); err != nil {
		t.Fatalf("SaveTeam() error = %v", err)
	}

	// Create local skills file
	local := &SkillsFile{
		Version: 1,
		Skills: map[string]Skill{
			"local-skill": {
				Source:      "personal/skills/local-skill",
				Commit:      "local456",
				Description: "A local skill",
			},
		},
	}
	if err := local.SaveLocal(dir); err != nil {
		t.Fatalf("SaveLocal() error = %v", err)
	}

	// Load team
	loadedTeam, err := LoadTeam(dir)
	if err != nil {
		t.Fatalf("LoadTeam() error = %v", err)
	}
	if _, ok := loadedTeam.Skills["team-skill"]; !ok {
		t.Error("LoadTeam() missing team-skill")
	}

	// Load local
	loadedLocal, err := LoadLocal(dir)
	if err != nil {
		t.Fatalf("LoadLocal() error = %v", err)
	}
	if _, ok := loadedLocal.Skills["local-skill"]; !ok {
		t.Error("LoadLocal() missing local-skill")
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()

	// Create team skills
	team := &SkillsFile{
		Version: 1,
		Skills: map[string]Skill{
			"shared-skill": {
				Source:      "org/skills/shared",
				Commit:      "team-version",
				Description: "Team version of shared skill",
			},
			"team-only": {
				Source:      "org/skills/team-only",
				Commit:      "abc",
				Description: "Team only skill",
			},
		},
	}
	if err := team.SaveTeam(dir); err != nil {
		t.Fatalf("SaveTeam() error = %v", err)
	}

	// Create local skills with override
	local := &SkillsFile{
		Version: 1,
		Skills: map[string]Skill{
			"shared-skill": {
				Source:      "personal/skills/shared",
				Commit:      "local-version",
				Description: "Local override of shared skill",
			},
			"local-only": {
				Source:      "personal/skills/local-only",
				Commit:      "def",
				Description: "Local only skill",
			},
		},
	}
	if err := local.SaveLocal(dir); err != nil {
		t.Fatalf("SaveLocal() error = %v", err)
	}

	// Load all
	all, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	// Should have 3 skills total
	if len(all) != 3 {
		t.Errorf("len(LoadAll()) = %d, want 3", len(all))
	}

	// shared-skill should be local version (local overrides team)
	if skill, ok := all["shared-skill"]; !ok {
		t.Error("LoadAll() missing shared-skill")
	} else if skill.Commit != "local-version" {
		t.Errorf("shared-skill.Commit = %q, want %q (local should override)", skill.Commit, "local-version")
	}

	// team-only should exist
	if _, ok := all["team-only"]; !ok {
		t.Error("LoadAll() missing team-only")
	}

	// local-only should exist
	if _, ok := all["local-only"]; !ok {
		t.Error("LoadAll() missing local-only")
	}
}

func TestLoadAllEmpty(t *testing.T) {
	dir := t.TempDir()

	// Load from empty directory (no skills files)
	all, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(all) != 0 {
		t.Errorf("len(LoadAll()) = %d, want 0", len(all))
	}
}

func TestSkillsFileAddRemoveGet(t *testing.T) {
	sf := &SkillsFile{Version: 1}

	// Add
	sf.Add("skill-1", Skill{
		Source:      "owner/skills/skill-1",
		Commit:      "abc",
		Description: "First skill",
	})

	// Get existing
	skill, ok := sf.Get("skill-1")
	if !ok {
		t.Error("Get(skill-1) returned false")
	}
	if skill.Source != "owner/skills/skill-1" {
		t.Errorf("Get() Source = %q, want %q", skill.Source, "owner/skills/skill-1")
	}

	// Get non-existing
	_, ok = sf.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) returned true")
	}

	// Remove existing
	if !sf.Remove("skill-1") {
		t.Error("Remove(skill-1) returned false")
	}
	if _, ok := sf.Get("skill-1"); ok {
		t.Error("Get(skill-1) returned true after Remove")
	}

	// Remove non-existing
	if sf.Remove("nonexistent") {
		t.Error("Remove(nonexistent) returned true")
	}
}

func TestSkillsFileNames(t *testing.T) {
	sf := &SkillsFile{
		Version: 1,
		Skills: map[string]Skill{
			"zebra":    {Source: "a/b/zebra"},
			"alpha":    {Source: "a/b/alpha"},
			"middle":   {Source: "a/b/middle"},
		},
	}

	names := sf.Names()
	if len(names) != 3 {
		t.Fatalf("len(Names()) = %d, want 3", len(names))
	}

	// Should be sorted
	expected := []string{"alpha", "middle", "zebra"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestSkillsFileNamesEmpty(t *testing.T) {
	sf := &SkillsFile{Version: 1}
	names := sf.Names()
	if names != nil && len(names) != 0 {
		t.Errorf("Names() = %v, want nil or empty", names)
	}
}

func TestSaveDefaultsVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills.yaml")

	// Create without version
	sf := &SkillsFile{
		Skills: map[string]Skill{
			"test": {Source: "a/b/test"},
		},
	}

	if err := sf.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Reload and check version was set
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1 (should default)", loaded.Version)
	}
}

func TestPaths(t *testing.T) {
	beadsDir := "/test/.beads"

	teamPath := TeamPath(beadsDir)
	if teamPath != "/test/.beads/skills.yaml" {
		t.Errorf("TeamPath() = %q, want %q", teamPath, "/test/.beads/skills.yaml")
	}

	localPath := LocalPath(beadsDir)
	if localPath != "/test/.beads/skills.local.yaml" {
		t.Errorf("LocalPath() = %q, want %q", localPath, "/test/.beads/skills.local.yaml")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skills.yaml")

	// Write invalid YAML
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("Load() expected error for invalid YAML")
	}
}
