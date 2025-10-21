package cmd

import (
	"fmt"
	"log"
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
	Run: func(cmd *cobra.Command, args []string) {
		projectKey, _ := cmd.Flags().GetString("project")

		if projectKey == "" {
			log.Fatal("Il flag --project è obbligatorio.")
		}

		client, err := jira.NewClient()
		if err != nil {
			log.Fatalf("Errore nella creazione del client Jira: %v", err)
		}

		// Selettore interattivo
		versionToUse := selectJiraVersion(client, projectKey)
		fmt.Printf("✅ Analisi repository per la versione: %s\n", versionToUse.Name)

		issues, err := jira.GetIssuesForVersion(client, projectKey, versionToUse.Name)
		if err != nil {
			log.Fatalf("Errore nel recupero dei ticket: %v", err)
		}

		if len(issues) == 0 {
			fmt.Println("⚠️  Nessun ticket trovato per questa versione.")
			return
		}

		// Mappa per raggruppare: etichetta -> lista di issue
		labelsToIssues := make(map[string][]jira.Issue)

		for _, issue := range issues {
			// Non mostriamo i sub-task in questa vista, solo i ticket "genitori"
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
			return
		}

		var sortedLabels []string
		for label := range labelsToIssues {
			sortedLabels = append(sortedLabels, label)
		}
		sort.Strings(sortedLabels)

		fmt.Printf("📂 Impatto sui Repository (raggruppato per etichetta):\n")

		// Stampa
		for _, label := range sortedLabels {
			fmt.Printf("\n%s\n", strings.Repeat("─", 80))
			fmt.Printf("🏷️  %s (%d issue)\n", label, len(labelsToIssues[label]))
			fmt.Printf("%s\n", strings.Repeat("─", 80))

			issuesInLabel := labelsToIssues[label]
			for _, issue := range issuesInLabel {
				fmt.Printf("  - [%s] %s (%s)\n", issue.Key, issue.Fields.Summary, issue.Fields.IssueType.Name)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(impactedReposCmd)
	impactedReposCmd.Flags().StringP("project", "p", "", "Chiave del progetto Jira (es. PROJ)")
}
