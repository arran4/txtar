package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// SkillList is a subcommand `txtar skill-list` -- List installed skills
//
// Flags:
//	format: --format (default: "text") Output format (text or json)
//	scope: --scope (default: "") Filter by scope (user or project)
//	agent: --agent (default: "") Filter by agent
func SkillList(format, scope, agent string) error {
	skills, err := getInstalledSkills(scope, agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting installed skills: %v\n", err)
		os.Exit(1)
	}

	if format == "json" {
		data, err := json.MarshalIndent(skills, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting json: %v\n", err)
			os.Exit(1)
		}
		if string(data) == "null" {
			fmt.Println("[]")
		} else {
			fmt.Println(string(data))
		}
		return nil
	}

	if len(skills) == 0 {
		fmt.Println("No skills installed.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSCOPE\tAGENT\tSOURCE\tREVISION")
	for _, s := range skills {
		agentName := s.Agent
		if agentName == "" {
			agentName = "common"
		}
		rev := s.Revision
		if len(rev) > 8 {
			rev = rev[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.Scope, agentName, s.Source, rev)
	}
	w.Flush()
	return nil
}
