package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

// skillsAddCmd installs a skill from GitHub
var skillsAddCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Install a skill from GitHub",
	Long: `Install a skill from a GitHub repository.

The source can be:
  - GitHub shorthand: owner/repo/path/to/skill
  - Full GitHub URL: https://github.com/owner/repo/tree/main/path/to/skill

The skill will be added to skills.yaml (team-shared) by default.
Use --local to install to skills.local.yaml (personal, git-ignored).

Examples:
  bd skills add trailofbits/skills/plugins/differential-review
  bd skills add https://github.com/anthropics/skills/tree/main/pdf-processing
  bd skills add vercel-labs/agent-skills/frontend-design --local`,
	Args: cobra.ExactArgs(1),
	Run:  runSkillsAdd,
}

var (
	skillsAddLocal bool   // --local flag
	skillsAddName  string // --name flag (override skill name)
)

func runSkillsAdd(cmd *cobra.Command, args []string) {
	source := args[0]

	beadsDir := beads.FindBeadsDir()
	if beadsDir == "" {
		FatalErrorRespectJSON("not in a beads project (no .beads directory found)")
	}

	// Find project root (parent of .beads)
	projectRoot := filepath.Dir(beadsDir)

	if !quietFlag {
		fmt.Printf("Fetching skill from %s...\n", source)
	}

	// Resolve and fetch the skill
	ctx := context.Background()
	parsedSource, result, err := skills.ResolveAndFetch(ctx, source)
	if err != nil {
		FatalErrorRespectJSON("fetching skill: %v", err)
	}

	// Determine skill name (from frontmatter or --name override)
	skillName := result.SkillMD.Frontmatter.Name
	if skillsAddName != "" {
		skillName = skillsAddName
	}

	// Load existing skills file
	var sf *skills.SkillsFile
	if skillsAddLocal {
		sf, err = skills.LoadLocal(beadsDir)
	} else {
		sf, err = skills.LoadTeam(beadsDir)
	}
	if err != nil {
		FatalErrorRespectJSON("loading skills file: %v", err)
	}

	// Check if skill already exists
	if existing, exists := sf.Get(skillName); exists {
		if existing.Commit == result.Commit {
			fmt.Printf("Skill %q already installed at same version (%s)\n", skillName, result.Commit[:7])
			return
		}
		if !quietFlag {
			fmt.Printf("Updating skill %q (%s -> %s)\n", skillName, existing.Commit[:7], result.Commit[:7])
		}
	}

	// Create skill entry
	skill := skills.Skill{
		Source:      parsedSource.String(),
		Commit:      result.Commit,
		Description: result.SkillMD.Frontmatter.Description,
		Tags:        result.SkillMD.Frontmatter.Tags,
		Size:        int64(len(result.Content)),
		InstalledAt: time.Now().UTC(),
		InstalledBy: getActor(),
	}

	// Add to skills file
	sf.Add(skillName, skill)

	// Save skills file
	if skillsAddLocal {
		if err := sf.SaveLocal(beadsDir); err != nil {
			FatalErrorRespectJSON("saving skills file: %v", err)
		}
	} else {
		if err := sf.SaveTeam(beadsDir); err != nil {
			FatalErrorRespectJSON("saving skills file: %v", err)
		}
	}

	// Cache the skill content to .claude/skills/<name>/SKILL.md
	cacheDir := filepath.Join(projectRoot, ".claude", "skills", skillName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		FatalErrorRespectJSON("creating cache directory: %v", err)
	}

	cachePath := filepath.Join(cacheDir, "SKILL.md")
	// #nosec G306 - SKILL.md is meant to be readable
	if err := os.WriteFile(cachePath, result.Content, 0644); err != nil {
		FatalErrorRespectJSON("caching skill content: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"name":        skillName,
			"source":      parsedSource.String(),
			"commit":      result.Commit,
			"description": result.SkillMD.Frontmatter.Description,
			"tags":        result.SkillMD.Frontmatter.Tags,
			"size":        len(result.Content),
			"scope":       scopeString(skillsAddLocal),
			"cache_path":  cachePath,
		})
		return
	}

	scope := "team"
	if skillsAddLocal {
		scope = "local"
	}
	fmt.Printf("Installed %s (%s, %s)\n", skillName, scope, result.Commit[:7])
}

func scopeString(local bool) string {
	if local {
		return "local"
	}
	return "team"
}

// skillsLoadCmd outputs the full content of an installed skill
var skillsLoadCmd = &cobra.Command{
	Use:   "load <name>",
	Short: "Output skill content for agent",
	Long: `Output the full content of an installed skill.

This command is primarily used by agents to load skill instructions on-demand.
The agent reads lightweight metadata at startup, then loads full content when needed.

The skill must be installed (via 'bd skills add') before it can be loaded.

Examples:
  bd skills load differential-review   # Output skill content
  bd skills load pdf-processing --json # Output as JSON`,
	Args: cobra.ExactArgs(1),
	Run:  runSkillsLoad,
}

func runSkillsLoad(cmd *cobra.Command, args []string) {
	skillName := args[0]

	beadsDir := beads.FindBeadsDir()
	if beadsDir == "" {
		FatalErrorRespectJSON("not in a beads project (no .beads directory found)")
	}

	// Find project root (parent of .beads)
	projectRoot := filepath.Dir(beadsDir)

	// Check if skill is installed (load both team and local)
	allSkills, err := skills.LoadAll(beadsDir)
	if err != nil {
		FatalErrorRespectJSON("loading skills: %v", err)
	}

	skill, exists := allSkills[skillName]
	if !exists {
		FatalErrorRespectJSON("skill %q not installed\n\nInstall with: bd skills add <source>", skillName)
	}

	// Read cached SKILL.md content
	cachePath := filepath.Join(projectRoot, ".claude", "skills", skillName, "SKILL.md")
	content, err := os.ReadFile(cachePath) // #nosec G304 - path constructed from validated skill name
	if err != nil {
		if os.IsNotExist(err) {
			FatalErrorRespectJSON("skill %q is installed but cached content is missing\n\nRe-sync with: bd skills sync", skillName)
		}
		FatalErrorRespectJSON("reading skill content: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"name":        skillName,
			"source":      skill.Source,
			"commit":      skill.Commit,
			"description": skill.Description,
			"tags":        skill.Tags,
			"content":     string(content),
			"size":        len(content),
		})
		return
	}

	// Output raw content for agent consumption
	fmt.Print(string(content))
}

func init() {
	// Register add subcommand with flags
	skillsAddCmd.Flags().BoolVar(&skillsAddLocal, "local", false, "Install to skills.local.yaml (personal, git-ignored)")
	skillsAddCmd.Flags().StringVar(&skillsAddName, "name", "", "Override skill name (default: from SKILL.md frontmatter)")
	skillsCmd.AddCommand(skillsAddCmd)

	// Register list subcommand
	skillsCmd.AddCommand(skillsListCmd)

	// Register load subcommand
	skillsCmd.AddCommand(skillsLoadCmd)

	// Register skills command
	rootCmd.AddCommand(skillsCmd)
}
