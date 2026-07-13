package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SkillMetadata holds provenance and version info for an installed skill.
type SkillMetadata struct {
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	Path        string    `json:"path"`
	Revision    string    `json:"revision"` // e.g. git commit hash or local hash
	Scope       string    `json:"scope"`
	Agent       string    `json:"agent"`
	InstalledAt time.Time `json:"installed_at"`
	AppVersion  string    `json:"app_version"`
}

const (
	metadataFileName = ".txtar-skill-metadata.json"
	skillFileName    = "SKILL.md"
)

func getAgentDir(scope, agent string) (string, error) {
	if scope != "user" && scope != "project" {
		return "", fmt.Errorf("invalid scope %q, must be 'user' or 'project'", scope)
	}

	// Default to a common .agents directory if no agent is specified
	agentName := agent
	if agentName == "" {
		agentName = "common"
	}

	var baseDir string
	if scope == "user" || scope == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home dir: %w", err)
		}
		baseDir = home
	} else {
		// Project scope
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}
		baseDir = cwd
	}

	// Conventional agent skill location: <base>/.agents/<agent>/skills
	return filepath.Join(baseDir, ".agents", agentName, "skills"), nil
}

func readMetadata(skillDir string) (*SkillMetadata, error) {
	metaPath := filepath.Join(skillDir, metadataFileName)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No metadata found
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var meta SkillMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	return &meta, nil
}

func writeMetadata(skillDir string, meta *SkillMetadata) error {
	metaPath := filepath.Join(skillDir, metadataFileName)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	return os.WriteFile(metaPath, data, 0644)
}

func getInstalledSkills(scope, agent string) ([]SkillMetadata, error) {
	var skills []SkillMetadata

	scopesToSearch := []string{scope}
	if scope == "" {
		scopesToSearch = []string{"user", "project"}
	}

	for _, s := range scopesToSearch {
		var agents []string
		if agent == "" {
			// Discover all agents
			baseDir := ""
			if s == "user" {
				home, _ := os.UserHomeDir()
				baseDir = home
			} else {
				baseDir, _ = os.Getwd()
			}
			agentsDir := filepath.Join(baseDir, ".agents")
		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return skills, nil
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				agents = append(agents, e.Name())
			}
		}
	} else {
		agents = []string{agent}
	}

		for _, a := range agents {
			agentDir, err := getAgentDir(s, a)
			if err != nil {
				continue
			}
			entries, err := os.ReadDir(agentDir)
			if err != nil {
				continue
			}

			for _, e := range entries {
				if e.IsDir() {
					skillDir := filepath.Join(agentDir, e.Name())
					meta, err := readMetadata(skillDir)
					if err == nil && meta != nil {
						skills = append(skills, *meta)
					}
				}
			}
		}
	}

	return skills, nil
}
