package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/beads"
	"github.com/steveyegge/beads/internal/skills"
)

// skillsCmd is the parent command for skills subcommands
var skillsCmd = &cobra.Command{
	Use:     "skills",
	GroupID: "setup",
	Short:   "Manage agent skills",
	Long: `Commands for managing agent skills.

Skills are reusable instructions that extend agent capabilities. They can be
shared across a team (skills.yaml) or kept personal (skills.local.yaml).

Examples:
  bd skills list              # List all installed skills
  bd skills add <source>      # Add a skill from GitHub
  bd skills remove <name>     # Remove an installed skill
  bd skills load <name>       # Output skill content for agent`,
}

// skillsListCmd lists all installed skills
var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	Long: `List all installed skills from both team and local configurations.

Skills from skills.local.yaml override team skills with the same name.

Examples:
  bd skills list         # List all skills
  bd skills list --json  # Output as JSON`,
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		beadsDir := beads.FindBeadsDir()
		if beadsDir == "" {
			FatalErrorRespectJSON("not in a beads project (no .beads directory found)")
		}

		// Load team skills
		teamSkills, err := skills.LoadTeam(beadsDir)
		if err != nil {
			FatalErrorRespectJSON("loading team skills: %v", err)
		}

		// Load local skills
		localSkills, err := skills.LoadLocal(beadsDir)
		if err != nil {
			FatalErrorRespectJSON("loading local skills: %v", err)
		}

		if jsonOutput {
			outputSkillsJSON(teamSkills, localSkills)
			return
		}

		outputSkillsTable(teamSkills, localSkills)
	},
}

// skillEntry represents a skill for display purposes
type skillEntry struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Scope       string   `json:"scope"` // "team" or "local"
}

func outputSkillsJSON(teamSkills, localSkills *skills.SkillsFile) {
	entries := []skillEntry{}

	// Add team skills
	for name, skill := range teamSkills.Skills {
		// Check if overridden by local
		scope := "team"
		if _, exists := localSkills.Skills[name]; exists {
			continue // Skip, will be added from local
		}
		entries = append(entries, skillEntry{
			Name:        name,
			Source:      skill.Source,
			Description: skill.Description,
			Tags:        skill.Tags,
			Scope:       scope,
		})
	}

	// Add local skills (these take precedence)
	for name, skill := range localSkills.Skills {
		entries = append(entries, skillEntry{
			Name:        name,
			Source:      skill.Source,
			Description: skill.Description,
			Tags:        skill.Tags,
			Scope:       "local",
		})
	}

	// Sort by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	outputJSON(map[string]interface{}{
		"skills": entries,
		"count":  len(entries),
	})
}

func outputSkillsTable(teamSkills, localSkills *skills.SkillsFile) {
	// Merge all skill names
	allNames := make(map[string]bool)
	for name := range teamSkills.Skills {
		allNames[name] = true
	}
	for name := range localSkills.Skills {
		allNames[name] = true
	}

	if len(allNames) == 0 {
		fmt.Println("No skills installed")
		fmt.Println("")
		fmt.Println("Add skills with: bd skills add <source>")
		return
	}

	// Sort names
	names := make([]string, 0, len(allNames))
	for name := range allNames {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Printf("Installed skills (%d):\n\n", len(names))

	for _, name := range names {
		// Prefer local over team
		var skill skills.Skill
		var scope string
		if s, exists := localSkills.Skills[name]; exists {
			skill = s
			scope = "local"
		} else {
			skill = teamSkills.Skills[name]
			scope = "team"
		}

		// Format output
		fmt.Printf("  %s", name)
		if scope == "local" {
			fmt.Printf(" (local)")
		}
		fmt.Println()

		if skill.Description != "" {
			// Truncate long descriptions
			desc := skill.Description
			if len(desc) > 70 {
				desc = desc[:67] + "..."
			}
			fmt.Printf("    %s\n", desc)
		}

		if len(skill.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(skill.Tags, ", "))
		}

		fmt.Println()
	}
}

func init() {
	// Register list subcommand
	skillsCmd.AddCommand(skillsListCmd)

	// Register skills command
	rootCmd.AddCommand(skillsCmd)
}
