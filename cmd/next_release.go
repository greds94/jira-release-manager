package cmd

import (
	"fmt"
	"log"
	"strings"

	"jira-release-manager/internal/jira"

	"github.com/spf13/cobra"
)

var nextReleaseCmd = &cobra.Command{
	Use:   "next-release",
	Short: "Mostra i ticket della prossima release pianificata per un progetto.",
	Long: `Recupera e visualizza tutti i ticket (inclusi i sub-task) pianificati 
per la prossima versione di rilascio non ancora pubblicata.`,
	Example: `  jira-release-manager next-release --project PROJ
  jira-release-manager next-release -p PROJ --detailed`,
	Run: func(cmd *cobra.Command, args []string) {
		projectKey, _ := cmd.Flags().GetString("project")
		detailed, _ := cmd.Flags().GetBool("detailed")

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

		releaseDate := "Non specificata"
		if nextVersion.ReleaseDate != "" {
			releaseDate = nextVersion.ReleaseDate
		}

		fmt.Printf("✅ Prossima release trovata: %s (Data: %s)\n", nextVersion.Name, releaseDate)
		if nextVersion.Description != "" {
			fmt.Printf("   Descrizione: %s\n", nextVersion.Description)
		}
		fmt.Println()

		issues, err := jira.GetIssuesForVersion(client, projectKey, nextVersion.Name)
		if err != nil {
			log.Fatalf("Errore nel recupero dei ticket: %v", err)
		}

		if len(issues) == 0 {
			fmt.Println("⚠️  Nessun ticket trovato per questa versione.")
			return
		}

		// Raggruppa i ticket per tipo
		parentIssues := make(map[string][]jira.Issue)
		subtaskMap := make(map[string][]jira.Issue)

		for _, issue := range issues {
			if issue.Fields.IssueType.Subtask {
				if issue.Fields.Parent != nil {
					parentKey := issue.Fields.Parent.Key
					subtaskMap[parentKey] = append(subtaskMap[parentKey], issue)
				}
			} else {
				issueType := issue.Fields.IssueType.Name
				parentIssues[issueType] = append(parentIssues[issueType], issue)
			}
		}

		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("  TICKET PIANIFICATI PER LA VERSIONE '%s'\n", nextVersion.Name)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		// Stampa i ticket raggruppati per tipo
		for issueType, issues := range parentIssues {
			fmt.Printf("📌 %s (%d)\n", strings.ToUpper(issueType), len(issues))
			fmt.Println(strings.Repeat("─", 80))

			for _, issue := range issues {
				assignee := "Non assegnato"
				if issue.Fields.Assignee != nil {
					assignee = issue.Fields.Assignee.DisplayName
				}

				statusIcon := getStatusIcon(issue.Fields.Status.Name)

				fmt.Printf("  %s [%s] %s\n", statusIcon, issue.Key, issue.Fields.Summary)

				if detailed {
					fmt.Printf("     Status: %s | Assignee: %s\n", issue.Fields.Status.Name, assignee)
					if issue.Fields.Priority != nil {
						fmt.Printf("     Priority: %s\n", issue.Fields.Priority.Name)
					}
				} else {
					fmt.Printf("     %s - %s\n", issue.Fields.Status.Name, assignee)
				}

				// Stampa i sub-task se presenti
				if subtasks, ok := subtaskMap[issue.Key]; ok && len(subtasks) > 0 {
					fmt.Printf("     Sub-tasks:\n")
					for _, subtask := range subtasks {
						subAssignee := "Non assegnato"
						if subtask.Fields.Assignee != nil {
							subAssignee = subtask.Fields.Assignee.DisplayName
						}
						subStatusIcon := getStatusIcon(subtask.Fields.Status.Name)
						fmt.Printf("       %s [%s] %s (%s - %s)\n",
							subStatusIcon, subtask.Key, subtask.Fields.Summary,
							subtask.Fields.Status.Name, subAssignee)
					}
				}
				fmt.Println()
			}
		}

		// Statistiche finali
		totalParent := 0
		totalSubtasks := 0
		for _, issues := range parentIssues {
			totalParent += len(issues)
		}
		for _, subtasks := range subtaskMap {
			totalSubtasks += len(subtasks)
		}

		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("  TOTALE: %d ticket principali, %d sub-task\n", totalParent, totalSubtasks)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	},
}

func getStatusIcon(status string) string {
	status = strings.ToLower(status)
	switch {
	case strings.Contains(status, "done") || strings.Contains(status, "closed") || strings.Contains(status, "resolved"):
		return "✅"
	case strings.Contains(status, "progress") || strings.Contains(status, "in progress"):
		return "🔄"
	case strings.Contains(status, "review"):
		return "👀"
	case strings.Contains(status, "todo") || strings.Contains(status, "open"):
		return "📋"
	case strings.Contains(status, "blocked"):
		return "🚫"
	default:
		return "•"
	}
}

func init() {
	rootCmd.AddCommand(nextReleaseCmd)
	nextReleaseCmd.Flags().StringP("project", "p", "", "Chiave del progetto Jira (es. PROJ)")
	nextReleaseCmd.Flags().BoolP("detailed", "d", false, "Mostra informazioni dettagliate per ogni ticket")
}
