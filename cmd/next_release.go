package cmd

import (
	"fmt"
	"log"
	"strings"

	"jira-release-manager/internal/jira"
	"jira-release-manager/internal/organizer"

	"github.com/spf13/cobra"
)

var nextReleaseCmd = &cobra.Command{
	Use:   "next-release",
	Short: "Mostra i ticket di una versione specifica.",
	Long: `Permette di selezionare interattivamente una versione e 
visualizza tutti i ticket (inclusi i sub-task) pianificati.`,
	Example: `  jira-release-manager next-release -p PROJ
  jira-release-manager next-release -p PROJ --detailed`,
	Run: func(cmd *cobra.Command, args []string) {
		projectKey, _ := cmd.Flags().GetString("project")
		detailed, _ := cmd.Flags().GetBool("detailed")
		debug, _ := cmd.Flags().GetBool("debug")

		if projectKey == "" {
			log.Fatal("Il flag --project è obbligatorio.")
		}

		client, err := jira.NewClient()
		if err != nil {
			log.Fatalf("Errore nella creazione del client Jira: %v", err)
		}

		versionToFetch := selectJiraVersion(client, projectKey)

		releaseDate := "Non specificata"
		if versionToFetch.ReleaseDate != "" {
			releaseDate = versionToFetch.ReleaseDate
		}

		fmt.Printf("✅ Release selezionata: %s (Data: %s)\n", versionToFetch.Name, releaseDate)
		if versionToFetch.Description != "" {
			fmt.Printf("   Descrizione: %s\n", versionToFetch.Description)
		}
		fmt.Println()

		// Usa il nome della versione trovata per recuperare le issue
		issues, err := jira.GetIssuesForVersion(client, projectKey, versionToFetch.Name)
		if err != nil {
			log.Fatalf("Errore nel recupero dei ticket: %v", err)
		}

		if len(issues) == 0 {
			fmt.Println("⚠️  Nessun ticket trovato per questa versione.")
			return
		}

		hierarchy := organizer.NewReleaseHierarchy(issues, debug)

		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("  TICKET PIANIFICATI PER LA VERSIONE '%s'\n", versionToFetch.Name)
		fmt.Printf("  (Esclusi i ticket completati)\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		// Contatori
		totalEpics := len(hierarchy.Epics)
		totalChildren := 0
		totalSubtasks := 0
		totalStandalone := 0

		printedSubtasks := make(map[string]bool)

		// Stampa gli Epic con le loro Story e sub-task
		if len(hierarchy.Epics) > 0 {
			fmt.Printf("📌 EPIC (%d)\n", len(hierarchy.Epics))
			fmt.Println(strings.Repeat("─", 80))

			for _, epic := range hierarchy.Epics {
				printIssue(epic, "", detailed)

				// Stampa le Story/Task dell'Epic
				if children, ok := hierarchy.EpicChildren[epic.Key]; ok && len(children) > 0 {
					totalChildren += len(children)
					for _, child := range children {
						printIssue(child, "  ", detailed)

						// Stampa i sub-task
						if subtasks, ok := hierarchy.SubtaskMap[child.Key]; ok && len(subtasks) > 0 {
							totalSubtasks += len(subtasks)
							for _, subtask := range subtasks {
								printIssue(subtask, "    ", detailed)
								printedSubtasks[subtask.Key] = true
							}
						}
					}
				}

				if subtasks, ok := hierarchy.SubtaskMap[epic.Key]; ok && len(subtasks) > 0 {
					totalSubtasks += len(subtasks)
					for _, subtask := range subtasks {
						printIssue(subtask, "  ", detailed)
						printedSubtasks[subtask.Key] = true
					}
				}

				fmt.Println()
			}
		}

		// Stampa gli altri ticket (non Epic) raggruppati per tipo
		for issueType, issues := range hierarchy.StandaloneIssues {
			fmt.Printf("📌 %s (%d)\n", strings.ToUpper(issueType), len(issues))
			fmt.Println(strings.Repeat("─", 80))
			totalStandalone += len(issues)

			for _, issue := range issues {
				printIssue(issue, "", detailed)

				// Stampa i sub-task
				if subtasks, ok := hierarchy.SubtaskMap[issue.Key]; ok && len(subtasks) > 0 {
					totalSubtasks += len(subtasks)
					for _, subtask := range subtasks {
						printIssue(subtask, "  ", detailed)
						printedSubtasks[subtask.Key] = true
					}
				}
				fmt.Println()
			}
		}

		var orphanedSubtasks []jira.Issue
		for _, subtasks := range hierarchy.SubtaskMap {
			for _, subtask := range subtasks {
				if _, printed := printedSubtasks[subtask.Key]; !printed {
					orphanedSubtasks = append(orphanedSubtasks, subtask)
				}
			}
		}

		if len(orphanedSubtasks) > 0 {
			fmt.Printf("📌 SUB-TASK AGGIUNTIVI (%d)\n", len(orphanedSubtasks))
			fmt.Printf("  (Ticket con fixVersion, ma genitore non in release o completato)\n")
			fmt.Println(strings.Repeat("─", 80))

			for _, subtask := range orphanedSubtasks {
				printIssue(subtask, "", detailed) // Stampa a livello root
				fmt.Println()
			}
			totalSubtasks += len(orphanedSubtasks) // Aggiungi al conteggio
		}

		// Statistiche finali (spostate alla fine)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		if totalEpics > 0 {
			fmt.Printf("  TOTALE: %d epic con %d issue figlie, %d issue standalone, %d sub-task\n",
				totalEpics, totalChildren, totalStandalone, totalSubtasks)
		} else {
			fmt.Printf("  TOTALE: %d issue, %d sub-task\n", totalStandalone, totalSubtasks)
		}
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	},
}

func printIssue(issue jira.Issue, indent string, detailed bool) {
	assignee := "Non assegnato"
	if issue.Fields.Assignee != nil {
		assignee = issue.Fields.Assignee.DisplayName
	}

	statusIcon := getStatusIcon(issue.Fields.Status.Name)

	// Determina il prefisso in base al livello di indentazione
	var prefix string
	if indent == "" {
		prefix = ""
	} else if indent == "  " {
		prefix = "├─ "
	} else if indent == "    " {
		prefix = "│  ├─ "
	} else {
		prefix = "└─ "
	}

	fmt.Printf("%s%s%s [%s] %s\n", indent, prefix, statusIcon, issue.Key, issue.Fields.Summary)

	detailIndent := indent
	if indent == "" {
		detailIndent = "   "
	} else if indent == "  " {
		detailIndent = "│  "
	} else {
		detailIndent = indent + "│  "
	}

	if detailed {
		fmt.Printf("%s    Status: %s | Assignee: %s\n", detailIndent, issue.Fields.Status.Name, assignee)
		if issue.Fields.Priority != nil {
			fmt.Printf("%s    Priority: %s\n", detailIndent, issue.Fields.Priority.Name)
		}
	} else {
		fmt.Printf("%s    %s - %s\n", detailIndent, issue.Fields.Status.Name, assignee)
	}
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
	nextReleaseCmd.Flags().Bool("debug", false, "Mostra informazioni di debug sulla gerarchia")
}
