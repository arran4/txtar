package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// performSkillUpdate runs the update logic for a specific skill based on its existing metadata.
func performSkillUpdate(skill *SkillMetadata, force bool) error {
	agentName := skill.Agent
	if agentName == "" {
		agentName = "common"
	}
	var baseDir string
	if skill.Scope == "user" {
		home, _ := os.UserHomeDir()
		baseDir = home
	} else {
		baseDir, _ = os.Getwd()
	}
	skillDir := filepath.Join(baseDir, ".agents", agentName, "skills", skill.Name)

	if !force {
		// Detect local modifications by hashing the directory
		currentHash, err := hashDir(skillDir)
		if err == nil && currentHash != skill.Revision && !strings.HasPrefix(skill.Source, "http") && !strings.Contains(skill.Source, "/") {
			// Local skills that have changed hash shouldn't be overwritten automatically without force
			// Remote skills might be tricky to compare if we only have the git commit hash.
			// The simplest check is if it's a local source and hash changed, complain.
			if !strings.HasPrefix(skill.Source, "http") && !strings.Contains(skill.Source, "/") {
				// Actually, if we use revision to store hash for local, and it doesn't match now, it was modified
				fmt.Printf("Warning: skill %q has local modifications; use --force to replace\n", skill.Name)
				return nil
			}
		}
	}

	tempDir, newRevision, err := resolveSource(skill.Source)
	if err != nil {
		return fmt.Errorf("failed to resolve source: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if newRevision == skill.Revision && !force {
		fmt.Printf("Skill %q is up to date.\n", skill.Name)
		return nil
	}

	skillPaths, err := findSkills(tempDir)
	if err != nil {
		return fmt.Errorf("failed to scan source: %w", err)
	}

	if len(skillPaths) == 0 {
		return fmt.Errorf("no SKILL.md found in source during update")
	}

	var selectedSkillPath string
	for _, p := range skillPaths {
		if filepath.Base(p) == skill.Name {
			selectedSkillPath = p
			break
		}
	}

	if selectedSkillPath == "" {
		// Maybe it was explicitly provided via a sub-path before?
		// If only one exists, we can assume it's the one.
		if len(skillPaths) == 1 {
			selectedSkillPath = skillPaths[0]
		} else {
			return fmt.Errorf("could not unambiguously find skill %q in updated source", skill.Name)
		}
	}

	// Remove old installation safely
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove old skill: %w", err)
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate skill dir: %w", err)
	}

	if err := copyDir(selectedSkillPath, skillDir); err != nil {
		return fmt.Errorf("failed to copy updated skill: %w", err)
	}

	skill.Revision = newRevision
	skill.InstalledAt = time.Now().UTC()

	if err := writeMetadata(skillDir, skill); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	fmt.Printf("Successfully updated skill %q to %s\n", skill.Name, newRevision)
	return nil
}
