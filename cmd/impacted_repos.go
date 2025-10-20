package cmd

import (
	"fmt"
	"log"
	"sort"

	"jira-release-manager/internal/jira"

	"github.com/spf13/cobra"
)

var impactedReposCmd = &cobra.Command{
	Use:   "impacted-repos",
	Short: "Mostra i repository impattati (etichette) dalla prossima release.",
	Long: `Recupera tutti i ticket della prossima release e ne estrae le etichette (labels)
per identificare i repository o i componenti impattati.`,
	Example: `  jira-release-manager impacted-repos --project PROJ`,
	Run: func(cmd *cobra.Command, args []string) {
		projectKey, _ := cmd.Flags().GetString("project")

		if projectKey == "" {
			log.Fatal("Il flag --project è obbligatorio.")
		}

		client, err := jira.NewClient()
		if err != nil {
			log.Fatalf("Errore nella creazione del client Jira: %v", err)
		}

		fmt.Printf("🔎 Ricerca della prossima release per il progetto %s...\n", projectKey)
		nextVersion, err := jira.FindNextReleaseVersion(client, projectKey)
		if err != nil {
			log.Fatalf("Errore: %v", err)
		}

		fmt.Printf("✅ Prossima release trovata: %s\n\n", nextVersion.Name)

		issues, err := jira.GetIssuesForVersion(client, projectKey, nextVersion.Name)
		if err != nil {
			log.Fatalf("Errore nel recupero dei ticket: %v", err)
		}

		if len(issues) == 0 {
			fmt.Println("⚠️  Nessun ticket trovato per questa versione.")
			return
		}

		repoLabels := make(map[string]bool)

		for _, issue := range issues {
			if len(issue.Fields.Labels) > 0 {
				for _, label := range issue.Fields.Labels {
					repoLabels[label] = true
				}
			}
		}

		if len(repoLabels) == 0 {
			fmt.Println("ℹ️ Nessuna etichetta (repository) trovata sui ticket della release.")
			return
		}

		var sortedLabels []string
		for label := range repoLabels {
			sortedLabels = append(sortedLabels, label)
		}
		sort.Strings(sortedLabels)

		fmt.Printf("📂 Repository/Componenti impattati (basato sulle etichette):\n")
		fmt.Println("────────────────────────────────────────────────────────")
		for _, label := range sortedLabels {
			fmt.Printf("- %s\n", label)
		}
	},
}

func init() {
	rootCmd.AddCommand(impactedReposCmd)
	impactedReposCmd.Flags().StringP("project", "p", "", "Chiave del progetto Jira (es. PROJ)")
}
