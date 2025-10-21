package cmd

import (
	"fmt"
	"sort"
	"strings"

	"jira-release-manager/internal/jira"

	"github.com/spf13/cobra"
)

var impactedReposCmd = &cobra.Command{
	Use:   "impacted-repos",
	Short: "Mostra le issue raggruppate per repository (etichetta).",
	Long: `Permette di selezionare interattivamente una versione e
mostra tutti i ticket raggruppati per etichetta (repository).
I ticket senza etichetta vengono ignorati.`,
	Example: `  jira-release-manager impacted-repos -p PROJ`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Selettore interattivo
		versionToUse, err := selectJiraVersion(jiraClient, projectKey)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Analisi repository per la versione: %s\n", versionToUse.Name)

		issues, err := jira.GetIssuesForVersion(jiraClient, projectKey, versionToUse.Name)
		if err != nil {
			return fmt.Errorf("errore nel recupero dei ticket: %w", err)
		}

		if len(issues) == 0 {
			fmt.Println("⚠️  Nessun ticket trovato per questa versione.")
			return nil
		}

		// Mappa per raggruppare: etichetta -> lista di issue
		labelsToIssues := make(map[string][]jira.Issue)

		for _, issue := range issues {
			if issue.Fields.IssueType.Subtask {
				continue
			}
			if len(issue.Fields.Labels) > 0 {
				for _, label := range issue.Fields.Labels {
					labelsToIssues[label] = append(labelsToIssues[label], issue)
				}
			}
		}

		if len(labelsToIssues) == 0 {
			fmt.Println("ℹ️ Nessun ticket con etichette trovato per questa release.")
			return nil
		}

		var sortedLabels []string
		for label := range labelsToIssues {
			sortedLabels = append(sortedLabels, label)
		}
		sort.Strings(sortedLabels)

		fmt.Printf("📂 Impatto sui Repository (raggruppato per etichetta):\n")

		for _, label := range sortedLabels {
			fmt.Printf("\n%s\n", strings.Repeat("─", 80))
			fmt.Printf("🏷️  %s (%d issue)\n", label, len(labelsToIssues[label]))
			fmt.Printf("%s\n", strings.Repeat("─", 80))

			issuesInLabel := labelsToIssues[label]
			for _, issue := range issuesInLabel {
				fmt.Printf("  - [%s] %s (%s)\n", issue.Key, issue.Fields.Summary, issue.Fields.IssueType.Name)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(impactedReposCmd)
}
