package organizer

import (
	"fmt"
	"strings"

	"jira-release-manager/internal/jira"
)

// ReleaseHierarchy detiene la struttura gerarchica organizzata dei ticket.
type ReleaseHierarchy struct {
	Epics            map[string]jira.Issue
	EpicChildren     map[string][]jira.Issue
	StandaloneIssues map[string][]jira.Issue
	SubtaskMap       map[string][]jira.Issue
}

// NewReleaseHierarchy organizza una lista piatta di issue in una gerarchia
// e restituisce una struct che la rappresenta.
func NewReleaseHierarchy(issues []jira.Issue, debug bool) *ReleaseHierarchy {
	h := &ReleaseHierarchy{
		Epics:            make(map[string]jira.Issue),
		EpicChildren:     make(map[string][]jira.Issue),
		StandaloneIssues: make(map[string][]jira.Issue),
		SubtaskMap:       make(map[string][]jira.Issue),
	}

	// Prima passata: identifica epics e subtask
	for _, issue := range issues {
		issueType := strings.ToLower(issue.Fields.IssueType.Name)

		if issue.Fields.IssueType.Subtask {
			if issue.Fields.Parent != nil {
				parentKey := issue.Fields.Parent.Key
				h.SubtaskMap[parentKey] = append(h.SubtaskMap[parentKey], issue)
			}
		} else if issueType == "epic" {
			h.Epics[issue.Key] = issue
		}
	}

	if debug {
		fmt.Println("\n🔍 DEBUG - Epic trovati:")
		for key := range h.Epics {
			fmt.Printf("  - %s\n", key)
		}
		fmt.Println()
	}

	// Seconda passata: organizza story/task sotto epic o come standalone
	for _, issue := range issues {
		if issue.Fields.IssueType.Subtask {
			continue // già gestiti
		}

		issueType := strings.ToLower(issue.Fields.IssueType.Name)
		if issueType == "epic" {
			continue // già gestiti
		}

		// Determina l'epic parent guardando sia epic che parent
		epicKey := ""

		// Priorità 1: campo Epic diretto
		if issue.Fields.Epic != nil && issue.Fields.Epic.Key != "" {
			epicKey = issue.Fields.Epic.Key
			if debug {
				fmt.Printf("🔍 DEBUG - %s ha Epic via campo 'epic': %s\n", issue.Key, epicKey)
			}
		}

		// Priorità 2: campo Parent se punta a un Epic
		if epicKey == "" && issue.Fields.Parent != nil && issue.Fields.Parent.Key != "" {
			parentKey := issue.Fields.Parent.Key
			// Verifica se il parent è un epic
			if _, isEpic := h.Epics[parentKey]; isEpic {
				epicKey = parentKey
				if debug {
					fmt.Printf("🔍 DEBUG - %s ha Epic via campo 'parent': %s\n", issue.Key, epicKey)
				}
			}
		}

		// Se ha un epic parent e l'epic è nella release
		if epicKey != "" {
			if _, epicExists := h.Epics[epicKey]; epicExists {
				h.EpicChildren[epicKey] = append(h.EpicChildren[epicKey], issue)
				if debug {
					fmt.Printf("🔍 DEBUG - %s aggiunto sotto epic %s\n", issue.Key, epicKey)
				}
				continue
			} else if debug {
				fmt.Printf("🔍 DEBUG - %s ha epic %s ma non è nella release\n", issue.Key, epicKey)
			}
		}

		// Non ha epic o epic non in release: standalone
		h.StandaloneIssues[issue.Fields.IssueType.Name] = append(h.StandaloneIssues[issue.Fields.IssueType.Name], issue)
		if debug {
			fmt.Printf("🔍 DEBUG - %s aggiunto come standalone\n", issue.Key)
		}
	}

	if debug {
		fmt.Println("\n🔍 DEBUG - Riepilogo:")
		fmt.Printf("  Epic: %d\n", len(h.Epics))
		for epicKey, children := range h.EpicChildren {
			fmt.Printf("  Epic %s ha %d figli\n", epicKey, len(children))
		}
		fmt.Println()
	}

	return h
}
