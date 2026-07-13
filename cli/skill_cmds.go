package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// findSkills searches for directories containing SKILL.md
func findSkills(root string) ([]string, error) {
	var skills []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == skillFileName {
			skills = append(skills, filepath.Dir(path))
		}
		return nil
	})
	return skills, err
}

// SkillInstall is a subcommand `txtar skill-install` -- Install an agent skill
//
// Flags:
//	scope: --scope (default: "user") Installation scope (user or project)
//	agent: --agent (default: "") Target agent (e.g., codex, claude, copilot, cursor)
//	source: @1 Source (e.g. owner/repo, or ./local-path)
//	nameOrPath: @2... Optional name or path within the source
func SkillInstall(scope string, agent string, source string, nameOrPath ...string) error {
	agentDir, err := getAgentDir(scope, agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	tempDir, revision, err := resolveSource(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving source: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	skillPaths, err := findSkills(tempDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning source for skills: %v\n", err)
		os.Exit(1)
	}

	if len(skillPaths) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no valid skill (SKILL.md) found in %s\n", source)
		os.Exit(1)
	}

	var selectedSkillPath string
	var skillName string

	targetNameOrPath := ""
	if len(nameOrPath) > 0 {
		targetNameOrPath = nameOrPath[0]
	}

	if targetNameOrPath != "" {
		for _, p := range skillPaths {
			// Check if name matches either the directory name or the relative path
			rel, _ := filepath.Rel(tempDir, p)
			if filepath.Base(p) == targetNameOrPath || rel == targetNameOrPath {
				selectedSkillPath = p
				skillName = filepath.Base(p)
				break
			}
		}
		if selectedSkillPath == "" {
			fmt.Fprintf(os.Stderr, "Error: skill %q was not found in %s\n", targetNameOrPath, source)
			os.Exit(1)
		}
	} else {
		if len(skillPaths) > 1 {
			var names []string
			for _, p := range skillPaths {
				names = append(names, filepath.Base(p))
			}
			fmt.Fprintf(os.Stderr, "Error: repository contains multiple skills: %s; specify one explicitly\n", strings.Join(names, ", "))
			os.Exit(1)
		}
		selectedSkillPath = skillPaths[0]
		skillName = filepath.Base(selectedSkillPath)
	}

	destDir := filepath.Join(agentDir, skillName)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(os.Stderr, "Error: skill %q is already installed. Use update to refresh it.\n", skillName)
		os.Exit(1)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating destination: %v\n", err)
		os.Exit(1)
	}

	if err := copyDir(selectedSkillPath, destDir); err != nil {
		os.RemoveAll(destDir)
		fmt.Fprintf(os.Stderr, "Error installing skill: %v\n", err)
		os.Exit(1)
	}

	relPath, _ := filepath.Rel(tempDir, selectedSkillPath)

	meta := &SkillMetadata{
		Name:        skillName,
		Source:      source,
		Path:        relPath,
		Revision:    revision,
		Scope:       scope,
		Agent:       agent,
		InstalledAt: time.Now().UTC(),
		AppVersion:  "dev", // Replace with real version if accessible
	}

	if err := writeMetadata(destDir, meta); err != nil {
		os.RemoveAll(destDir)
		fmt.Fprintf(os.Stderr, "Error writing metadata: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully installed skill %q to %s\n", skillName, destDir)
	return nil
}

// SkillList is a subcommand `txtar skill-list` -- List installed skills
//
// Flags:
//	format: --format (default: "text") Output format (text or json)
// SkillUpdate is a subcommand `txtar skill-update` -- Update an installed skill
//
// Flags:
//	all: --all (default: false) Update all skills
//	force: --force (default: false) Force update, overriding local modifications
//	name: @1... Skill name to update
func SkillUpdate(all bool, force bool, name ...string) error {
	if all {
		skills, err := getInstalledSkills("", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, s := range skills {
			if err := performSkillUpdate(&s, force); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating %q: %v\n", s.Name, err)
			}
		}
		return nil
	}
	targetName := ""
	if len(name) > 0 {
		targetName = name[0]
	}
	if targetName == "" {
		fmt.Fprintf(os.Stderr, "Error: missing skill name\n")
		os.Exit(1)
	}
	skills, err := getInstalledSkills("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	var target *SkillMetadata
	for _, s := range skills {
		if s.Name == targetName {
			target = &s
			break
		}
	}
	if target == nil {
		fmt.Fprintf(os.Stderr, "Error: skill %q not found\n", targetName)
		os.Exit(1)
	}
	if err := performSkillUpdate(target, force); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

// SkillRemove is a subcommand `txtar skill-remove` -- Remove an installed skill
//
// Flags:
//	name: @1 Skill name to remove
func SkillRemove(name string) error {
	targetName := name
	if targetName == "" {
		fmt.Fprintf(os.Stderr, "Error: missing skill name\n")
		os.Exit(1)
	}

	skills, err := getInstalledSkills("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var toRemove *SkillMetadata
	var toRemoveDir string
	for _, s := range skills {
		if s.Name == targetName {
			toRemove = &s
			agentName := s.Agent
			if agentName == "" {
				agentName = "common"
			}
			var baseDir string
			if s.Scope == "user" {
				home, _ := os.UserHomeDir()
				baseDir = home
			} else {
				baseDir, _ = os.Getwd()
			}
			toRemoveDir = filepath.Join(baseDir, ".agents", agentName, "skills", s.Name)
			break
		}
	}

	if toRemove == nil {
		fmt.Fprintf(os.Stderr, "Error: skill %q not found\n", targetName)
		os.Exit(1)
	}

	if err := os.RemoveAll(toRemoveDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing skill: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Removed skill %q from %s scope (agent: %s)\n", name, toRemove.Scope, toRemove.Agent)
	return nil
}

// SkillInspect is a subcommand `txtar skill-inspect` -- Inspect an installed skill
//
// Flags:
//	name: @1 Skill name to inspect
func SkillInspect(name string) error {
	targetName := name
	if targetName == "" {
		fmt.Fprintf(os.Stderr, "Error: missing skill name\n")
		os.Exit(1)
	}

	skills, err := getInstalledSkills("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var toInspect *SkillMetadata
	var toInspectDir string
	for _, s := range skills {
		if s.Name == targetName {
			toInspect = &s
			agentName := s.Agent
			if agentName == "" {
				agentName = "common"
			}
			var baseDir string
			if s.Scope == "user" {
				home, _ := os.UserHomeDir()
				baseDir = home
			} else {
				baseDir, _ = os.Getwd()
			}
			toInspectDir = filepath.Join(baseDir, ".agents", agentName, "skills", s.Name)
			break
		}
	}

	if toInspect == nil {
		fmt.Fprintf(os.Stderr, "Error: skill %q not found\n", targetName)
		os.Exit(1)
	}

	b, err := json.MarshalIndent(toInspect, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting metadata: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Metadata:")
	fmt.Println(string(b))
	fmt.Println("\nFiles:")

	filepath.Walk(toInspectDir, func(path string, info os.FileInfo, err error) error {
		if path == toInspectDir {
			return nil
		}
		rel, _ := filepath.Rel(toInspectDir, path)
		fmt.Println("  " + rel)
		return nil
	})

	return nil
}
