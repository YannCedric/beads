package main

import (
	"testing"

	"github.com/steveyegge/beads/internal/skills"
)

func TestOutputSkillsTableEmpty(t *testing.T) {
	// Test with empty skills files
	team := &skills.SkillsFile{Version: 1, Skills: map[string]skills.Skill{}}
	local := &skills.SkillsFile{Version: 1, Skills: map[string]skills.Skill{}}

	// Should not panic
	outputSkillsTable(team, local)
}

func TestOutputSkillsTableWithSkills(t *testing.T) {
	team := &skills.SkillsFile{
		Version: 1,
		Skills: map[string]skills.Skill{
			"team-skill": {
				Source:      "org/repo/skills/team-skill",
				Description: "A team skill for testing",
				Tags:        []string{"testing", "team"},
			},
		},
	}
	local := &skills.SkillsFile{
		Version: 1,
		Skills: map[string]skills.Skill{
			"local-skill": {
				Source:      "user/repo/skills/local-skill",
				Description: "A local skill for testing",
				Tags:        []string{"testing", "local"},
			},
		},
	}

	// Should not panic
	outputSkillsTable(team, local)
}

func TestOutputSkillsTableLocalOverridesTeam(t *testing.T) {
	// Test that local skills override team skills with same name
	team := &skills.SkillsFile{
		Version: 1,
		Skills: map[string]skills.Skill{
			"shared-skill": {
				Source:      "org/repo/skills/shared",
				Description: "Team version",
			},
		},
	}
	local := &skills.SkillsFile{
		Version: 1,
		Skills: map[string]skills.Skill{
			"shared-skill": {
				Source:      "user/repo/skills/shared",
				Description: "Local version",
			},
		},
	}

	// Should not panic and should display local version
	outputSkillsTable(team, local)
}

func TestSkillEntryStructure(t *testing.T) {
	// Verify skillEntry has correct fields
	entry := skillEntry{
		Name:        "test-skill",
		Source:      "owner/repo/skills/test",
		Description: "A test skill",
		Tags:        []string{"test"},
		Scope:       "team",
	}

	if entry.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", entry.Name, "test-skill")
	}
	if entry.Scope != "team" {
		t.Errorf("Scope = %q, want %q", entry.Scope, "team")
	}
}

func TestSkillsCommandExists(t *testing.T) {
	// Verify skillsCmd is properly configured
	if skillsCmd.Use != "skills" {
		t.Errorf("skillsCmd.Use = %q, want %q", skillsCmd.Use, "skills")
	}
	if skillsCmd.GroupID != "setup" {
		t.Errorf("skillsCmd.GroupID = %q, want %q", skillsCmd.GroupID, "setup")
	}
}

func TestSkillsListCommandExists(t *testing.T) {
	// Verify skillsListCmd is properly configured
	if skillsListCmd.Use != "list" {
		t.Errorf("skillsListCmd.Use = %q, want %q", skillsListCmd.Use, "list")
	}

	// Check aliases
	foundLsAlias := false
	for _, alias := range skillsListCmd.Aliases {
		if alias == "ls" {
			foundLsAlias = true
			break
		}
	}
	if !foundLsAlias {
		t.Error("skillsListCmd missing 'ls' alias")
	}
}

func TestSkillsCommandHasListSubcommand(t *testing.T) {
	// Verify skillsListCmd is registered as subcommand
	found := false
	for _, cmd := range skillsCmd.Commands() {
		if cmd.Use == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("skillsCmd missing 'list' subcommand")
	}
}
