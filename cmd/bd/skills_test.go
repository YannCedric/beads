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

func TestSkillsAddCommandExists(t *testing.T) {
	// Verify skillsAddCmd is properly configured
	if skillsAddCmd.Use != "add <source>" {
		t.Errorf("skillsAddCmd.Use = %q, want %q", skillsAddCmd.Use, "add <source>")
	}
}

func TestSkillsCommandHasAddSubcommand(t *testing.T) {
	// Verify skillsAddCmd is registered as subcommand
	found := false
	for _, cmd := range skillsCmd.Commands() {
		if cmd.Use == "add <source>" {
			found = true
			break
		}
	}
	if !found {
		t.Error("skillsCmd missing 'add' subcommand")
	}
}

func TestSkillsAddCommandFlags(t *testing.T) {
	// Verify --local flag exists
	localFlag := skillsAddCmd.Flags().Lookup("local")
	if localFlag == nil {
		t.Error("skillsAddCmd missing --local flag")
	} else if localFlag.DefValue != "false" {
		t.Errorf("--local default = %q, want %q", localFlag.DefValue, "false")
	}

	// Verify --name flag exists
	nameFlag := skillsAddCmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("skillsAddCmd missing --name flag")
	} else if nameFlag.DefValue != "" {
		t.Errorf("--name default = %q, want %q", nameFlag.DefValue, "")
	}
}

func TestScopeString(t *testing.T) {
	tests := []struct {
		local bool
		want  string
	}{
		{false, "team"},
		{true, "local"},
	}

	for _, tt := range tests {
		got := scopeString(tt.local)
		if got != tt.want {
			t.Errorf("scopeString(%v) = %q, want %q", tt.local, got, tt.want)
		}
	}
}

func TestSkillsLoadCommandExists(t *testing.T) {
	// Verify skillsLoadCmd is properly configured
	if skillsLoadCmd.Use != "load <name>" {
		t.Errorf("skillsLoadCmd.Use = %q, want %q", skillsLoadCmd.Use, "load <name>")
	}
}

func TestSkillsCommandHasLoadSubcommand(t *testing.T) {
	// Verify skillsLoadCmd is registered as subcommand
	found := false
	for _, cmd := range skillsCmd.Commands() {
		if cmd.Use == "load <name>" {
			found = true
			break
		}
	}
	if !found {
		t.Error("skillsCmd missing 'load' subcommand")
	}
}

func TestSkillsLoadCommandRequiresArg(t *testing.T) {
	// Verify the command requires exactly one argument
	if skillsLoadCmd.Args == nil {
		t.Error("skillsLoadCmd.Args is nil, expected ExactArgs(1)")
	}
}
