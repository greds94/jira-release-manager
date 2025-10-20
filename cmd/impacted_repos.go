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
	Long: `Recupera tutti i ticket per una versione e li raggruppa
per etichetta, per mostrare l'impatto su ogni repository.
Se --version non è specificato, usa la prossima release.`,
	Example: `  jira-release-manager impacted-repos -p PROJ
  jira-release-manager impacted-repos -p PROJ --version "Release 1.2.3"`,
	Run: func(cmd *cobra.Command, args []string) {
		projectKey, _ := cmd.Flags().GetString("project")
		versionName, _ := cmd.Flags().GetString("version")

		if projectKey == "" {
			log.Fatal("Il flag --project è obbligatorio.")
		}

		client, err := jira.NewClient()
		if err != nil {
			log.Fatalf("Errore nella creazione del client Jira: %v", err)
		}

		// Se la versione non è specificata, trova la prossima
		if versionName == "" {
			fmt.Printf("🔎 Ricerca della prossima release per il progetto %s...\n", projectKey)
			nextVersion, err := jira.FindNextReleaseVersion(client, projectKey)
			if err != nil {
				log.Fatalf("Errore: %v", err)
			}
			versionName = nextVersion.Name
			fmt.Printf("✅ Prossima release trovata: %s\n\n", versionName)
		} else {
			fmt.Printf("✅ Utilizzo della versione specificata: %s\n\n", versionName)
		}

		issues, err := jira.GetIssuesForVersion(client, projectKey, versionName)
		if err != nil {
			log.Fatalf("Errore nel recupero dei ticket: %v", err)
		}

		if len(issues) == 0 {
			fmt.Println("⚠️  Nessun ticket trovato per questa versione.")
			return
		}

		// Mappa per raggruppare: etichetta -> lista di issue
		labelsToIssues := make(map[string][]jira.Issue)
		const noLabelKey = "SENZA_ETICHETTA"

		for _, issue := range issues {
			// Non mostriamo i sub-task in questa vista, solo i ticket "genitori"
			if issue.Fields.IssueType.Subtask {
				continue
			}

			if len(issue.Fields.Labels) == 0 {
				labelsToIssues[noLabelKey] = append(labelsToIssues[noLabelKey], issue)
			} else {
				for _, label := range issue.Fields.Labels {
					labelsToIssues[label] = append(labelsToIssues[label], issue)
				}
			}
		}

		if len(labelsToIssues) == 0 {
			fmt.Println("ℹ️ Nessun ticket (non sub-task) trovato per la release.")
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
			if label == noLabelKey {
				fmt.Printf("🏷️  %s (%d issue)\n", noLabelKey, len(labelsToIssues[label]))
			} else {
				fmt.Printf("🏷️  %s (%d issue)\n", label, len(labelsToIssues[label]))
			}
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
	impactedReposCmd.Flags().StringP("version", "v", "", "Nome esatto della versione Jira (opzionale)")
}
