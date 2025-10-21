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

		// Organizza i ticket in una struttura gerarchica
		epics := make(map[string]jira.Issue)
		epicChildren := make(map[string][]jira.Issue)     // Story/Task sotto epic
		standaloneIssues := make(map[string][]jira.Issue) // Issue senza epic
		subtaskMap := make(map[string][]jira.Issue)

		// Prima passata: identifica epics e subtask
		for _, issue := range issues {
			issueType := strings.ToLower(issue.Fields.IssueType.Name)

			if issue.Fields.IssueType.Subtask {
				if issue.Fields.Parent != nil {
					parentKey := issue.Fields.Parent.Key
					subtaskMap[parentKey] = append(subtaskMap[parentKey], issue)
				}
			} else if issueType == "epic" {
				epics[issue.Key] = issue
			}
		}

		if debug {
			fmt.Println("\n🔍 DEBUG - Epic trovati:")
			for key := range epics {
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
				if _, isEpic := epics[parentKey]; isEpic {
					epicKey = parentKey
					if debug {
						fmt.Printf("🔍 DEBUG - %s ha Epic via campo 'parent': %s\n", issue.Key, epicKey)
					}
				}
			}

			// Se ha un epic parent e l'epic è nella release
			if epicKey != "" {
				if _, epicExists := epics[epicKey]; epicExists {
					epicChildren[epicKey] = append(epicChildren[epicKey], issue)
					if debug {
						fmt.Printf("🔍 DEBUG - %s aggiunto sotto epic %s\n", issue.Key, epicKey)
					}
					continue
				} else if debug {
					fmt.Printf("🔍 DEBUG - %s ha epic %s ma non è nella release\n", issue.Key, epicKey)
				}
			}

			// Non ha epic o epic non in release: standalone
			standaloneIssues[issue.Fields.IssueType.Name] = append(standaloneIssues[issue.Fields.IssueType.Name], issue)
			if debug {
				fmt.Printf("🔍 DEBUG - %s aggiunto come standalone\n", issue.Key)
			}
		}

		if debug {
			fmt.Println("\n🔍 DEBUG - Riepilogo:")
			fmt.Printf("  Epic: %d\n", len(epics))
			for epicKey, children := range epicChildren {
				fmt.Printf("  Epic %s ha %d figli\n", epicKey, len(children))
			}
			fmt.Println()
		}

		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("  TICKET PIANIFICATI PER LA VERSIONE '%s'\n", versionToFetch.Name)
		fmt.Printf("  (Esclusi i ticket completati)\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		// Contatori
		totalEpics := len(epics)
		totalChildren := 0
		totalSubtasks := 0
		totalStandalone := 0

		// Stampa gli Epic con le loro Story e sub-task
		if len(epics) > 0 {
			fmt.Printf("📌 EPIC (%d)\n", len(epics))
			fmt.Println(strings.Repeat("─", 80))

			for _, epic := range epics {
				printIssue(epic, "", detailed)

				// Stampa le Story/Task dell'Epic
				if children, ok := epicChildren[epic.Key]; ok && len(children) > 0 {
					totalChildren += len(children)
					for _, child := range children {
						printIssue(child, "  ", detailed)

						// Stampa i sub-task
						if subtasks, ok := subtaskMap[child.Key]; ok && len(subtasks) > 0 {
							totalSubtasks += len(subtasks)
							for _, subtask := range subtasks {
								printIssue(subtask, "    ", detailed)
							}
						}
					}
				}

				// Sub-task diretti dell'epic (raro ma possibile)
				if subtasks, ok := subtaskMap[epic.Key]; ok && len(subtasks) > 0 {
					totalSubtasks += len(subtasks)
					for _, subtask := range subtasks {
						printIssue(subtask, "  ", detailed)
					}
				}

				fmt.Println()
			}
		}

		// Stampa gli altri ticket (non Epic) raggruppati per tipo
		for issueType, issues := range standaloneIssues {
			fmt.Printf("📌 %s (%d)\n", strings.ToUpper(issueType), len(issues))
			fmt.Println(strings.Repeat("─", 80))
			totalStandalone += len(issues)

			for _, issue := range issues {
				printIssue(issue, "", detailed)

				// Stampa i sub-task
				if subtasks, ok := subtaskMap[issue.Key]; ok && len(subtasks) > 0 {
					totalSubtasks += len(subtasks)
					for _, subtask := range subtasks {
						printIssue(subtask, "  ", detailed)
					}
				}
				fmt.Println()
			}
		}

		// Statistiche finali
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
