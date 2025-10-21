package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"jira-release-manager/internal/jira"
	"jira-release-manager/internal/organizer"

	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Genera un changelog in formato Markdown per una versione.",
	Long: `Permette di selezionare interattivamente una versione e genera un changelog 
formattato in Markdown (o altri formati) basato sui ticket.`,
	Example: `  jira-release-manager changelog --project PROJ
  jira-release-manager changelog -p PROJ --output CHANGELOG.md
  jira-release-manager changelog -p PROJ --format slack`,
	Run: func(cmd *cobra.Command, args []string) {
		projectKey, _ := cmd.Flags().GetString("project")
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		includeSubtasks, _ := cmd.Flags().GetBool("include-subtasks")

		if projectKey == "" {
			log.Fatal("Il flag --project è obbligatorio.")
		}

		client, err := jira.NewClient()
		if err != nil {
			log.Fatalf("Errore: %v", err)
		}

		// Utilizza il selettore interattivo
		versionToFetch := selectJiraVersion(client, projectKey)
		fmt.Printf("✅ Generazione changelog per la versione: %s\n", versionToFetch.Name)

		issues, err := jira.GetIssuesForVersion(client, projectKey, versionToFetch.Name)
		if err != nil {
			log.Fatalf("Errore nel recupero dei ticket: %v", err)
		}

		hierarchy := organizer.NewReleaseHierarchy(issues, false)

		var changelog string
		switch format {
		case "markdown", "md":
			changelog = generateMarkdownChangelog(versionToFetch, hierarchy, includeSubtasks, client.BaseURL)
		case "slack":
			changelog = generateSlackChangelog(versionToFetch, hierarchy, includeSubtasks, client.BaseURL)
		case "html":
			changelog = generateHTMLChangelog(versionToFetch, hierarchy, includeSubtasks, client.BaseURL)
		default:
			changelog = generateMarkdownChangelog(versionToFetch, hierarchy, includeSubtasks, client.BaseURL)
		}

		if outputFile != "" {
			err := os.WriteFile(outputFile, []byte(changelog), 0644)
			if err != nil {
				log.Fatalf("Errore nel salvataggio del file: %v", err)
			}
			fmt.Printf("✅ Changelog salvato in: %s\n", outputFile)
		} else {
			fmt.Println(changelog)
		}
	},
}

func generateMarkdownChangelog(version *jira.Version, hierarchy *organizer.ReleaseHierarchy, includeSubtasks bool, baseURL string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 📋 Changelog - Versione %s\n\n", version.Name))

	releaseDate := time.Now().Format("2006-01-02")
	if version.ReleaseDate != "" {
		releaseDate = version.ReleaseDate
	}
	sb.WriteString(fmt.Sprintf("**Data di rilascio**: %s\n\n", releaseDate))

	if version.Description != "" {
		sb.WriteString(fmt.Sprintf("**Descrizione**: %s\n\n", version.Description))
	}

	sb.WriteString("---\n\n")

	// Usiamo direttamente le mappe dalla struct 'hierarchy'.
	epics := hierarchy.Epics
	epicChildren := hierarchy.EpicChildren
	standaloneIssues := hierarchy.StandaloneIssues
	subtaskMap := hierarchy.SubtaskMap

	printedSubtasks := make(map[string]bool)

	// Genera output
	if len(epics) > 0 {
		sb.WriteString("## 🎯 Epic\n\n")
		for _, epic := range epics {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, epic.Key)
			sb.WriteString(fmt.Sprintf("### **[%s](%s)** %s\n\n", epic.Key, issueURL, epic.Fields.Summary))

			if children, ok := epicChildren[epic.Key]; ok && len(children) > 0 {
				for _, child := range children {
					childURL := fmt.Sprintf("%s/browse/%s", baseURL, child.Key)
					sb.WriteString(fmt.Sprintf("- **[%s](%s)**: %s\n", child.Key, childURL, child.Fields.Summary))

					if includeSubtasks {
						if subtasks, ok := subtaskMap[child.Key]; ok {
							for _, subtask := range subtasks {
								subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
								sb.WriteString(fmt.Sprintf("  - [%s](%s): %s\n", subtask.Key, subtaskURL, subtask.Fields.Summary))
								printedSubtasks[subtask.Key] = true
							}
						}
					}
				}
				sb.WriteString("\n")
			}
		}
	}

	// Altri tipi
	preferredOrder := []string{"Story", "Task", "Improvement", "Bug"}
	typeEmoji := map[string]string{
		"Story":       "✨",
		"Task":        "📝",
		"Improvement": "🔧",
		"Bug":         "🐛",
	}

	for _, issueType := range preferredOrder {
		issuesList, ok := standaloneIssues[issueType]
		if !ok || len(issuesList) == 0 {
			continue
		}

		emoji := typeEmoji[issueType]
		if emoji == "" {
			emoji = "•"
		}

		sb.WriteString(fmt.Sprintf("## %s %s\n\n", emoji, issueType))

		for _, issue := range issuesList {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, issue.Key)
			sb.WriteString(fmt.Sprintf("- **[%s](%s)**: %s\n", issue.Key, issueURL, issue.Fields.Summary))

			if includeSubtasks {
				if subtasks, ok := subtaskMap[issue.Key]; ok {
					for _, subtask := range subtasks {
						subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
						sb.WriteString(fmt.Sprintf("  - [%s](%s): %s\n", subtask.Key, subtaskURL, subtask.Fields.Summary))
						printedSubtasks[subtask.Key] = true
					}
				}
			}
		}
		sb.WriteString("\n")
	}

	// Altri tipi non previsti
	for issueType, issuesList := range standaloneIssues {
		found := false
		for _, preferred := range preferredOrder {
			if issueType == preferred {
				found = true
				break
			}
		}
		if !found && len(issuesList) > 0 {
			sb.WriteString(fmt.Sprintf("## • %s\n\n", issueType))
			for _, issue := range issuesList {
				issueURL := fmt.Sprintf("%s/browse/%s", baseURL, issue.Key)
				sb.WriteString(fmt.Sprintf("- **[%s](%s)**: %s\n", issue.Key, issueURL, issue.Fields.Summary))
			}
			sb.WriteString("\n")
		}
	}

	var orphanedSubtasks []jira.Issue
	if includeSubtasks {
		for _, subtasks := range subtaskMap {
			for _, subtask := range subtasks {
				if _, printed := printedSubtasks[subtask.Key]; !printed {
					orphanedSubtasks = append(orphanedSubtasks, subtask)
				}
			}
		}
	}

	if len(orphanedSubtasks) > 0 {
		sb.WriteString("## 📎 Sub-task Aggiuntivi\n\n")
		sb.WriteString("*(Ticket con fixVersion, ma genitore non in questa release o completato)*\n\n")
		for _, subtask := range orphanedSubtasks {
			subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
			sb.WriteString(fmt.Sprintf("- [%s](%s): %s\n", subtask.Key, subtaskURL, subtask.Fields.Summary))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateSlackChangelog(version *jira.Version, hierarchy *organizer.ReleaseHierarchy, includeSubtasks bool, baseURL string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*📋 Changelog - Versione %s*\n\n", version.Name))

	releaseDate := time.Now().Format("2006-01-02")
	if version.ReleaseDate != "" {
		releaseDate = version.ReleaseDate
	}
	sb.WriteString(fmt.Sprintf("*Data di rilascio*: %s\n\n", releaseDate))

	epics := hierarchy.Epics
	epicChildren := hierarchy.EpicChildren
	standaloneIssues := hierarchy.StandaloneIssues
	// subtaskMap := hierarchy.SubtaskMap // Non usato in slack (includeSubtasks non implementato qui)

	if len(epics) > 0 {
		sb.WriteString("*🎯 Epic*\n")
		for _, epic := range epics {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, epic.Key)
			sb.WriteString(fmt.Sprintf("*<%s|%s>* %s\n", issueURL, epic.Key, epic.Fields.Summary))

			if children, ok := epicChildren[epic.Key]; ok {
				for _, child := range children {
					childURL := fmt.Sprintf("%s/browse/%s", baseURL, child.Key)
					sb.WriteString(fmt.Sprintf("  • <%s|%s>: %s\n", childURL, child.Key, child.Fields.Summary))
				}
			}
		}
		sb.WriteString("\n")
	}

	typeEmoji := map[string]string{
		"Story":       ":sparkles:",
		"Task":        ":memo:",
		"Improvement": ":wrench:",
		"Bug":         ":bug:",
	}

	for issueType, issuesList := range standaloneIssues {
		if len(issuesList) == 0 {
			continue
		}

		emoji := typeEmoji[issueType]
		if emoji == "" {
			emoji = ":small_blue_diamond:"
		}

		sb.WriteString(fmt.Sprintf("*%s %s*\n", emoji, issueType))

		for _, issue := range issuesList {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, issue.Key)
			sb.WriteString(fmt.Sprintf("• <%s|%s>: %s\n", issueURL, issue.Key, issue.Fields.Summary))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateHTMLChangelog(version *jira.Version, hierarchy *organizer.ReleaseHierarchy, includeSubtasks bool, baseURL string) string {
	var sb strings.Builder

	sb.WriteString("<html><head><meta charset=\"UTF-8\"><title>Changelog</title>")
	sb.WriteString("<style>")
	sb.WriteString("body{font-family:Arial,sans-serif;max-width:900px;margin:40px auto;padding:20px;line-height:1.6;}")
	sb.WriteString("h1{color:#0052CC;border-bottom:3px solid #0052CC;padding-bottom:10px;}")
	sb.WriteString("h2{color:#333;border-bottom:2px solid #0052CC;padding-bottom:5px;margin-top:30px;}")
	sb.WriteString("h3{color:#0052CC;margin-top:20px;font-size:1.1em;}")
	sb.WriteString("ul{list-style-type:none;padding-left:0;}")
	sb.WriteString("ul ul{padding-left:30px;}")
	sb.WriteString("li{margin:8px 0;}")
	sb.WriteString("a{color:#0052CC;text-decoration:none;font-weight:bold;}")
	sb.WriteString("a:hover{text-decoration:underline;}")
	sb.WriteString(".meta{color:#666;font-size:14px;margin-bottom:20px;}")
	sb.WriteString(".epic-box{background:#f4f5f7;padding:15px;margin:10px 0;border-left:4px solid #0052CC;}")
	sb.WriteString("</style></head><body>")

	sb.WriteString(fmt.Sprintf("<h1>📋 Changelog - Versione %s</h1>", version.Name))

	releaseDate := time.Now().Format("2006-01-02")
	if version.ReleaseDate != "" {
		releaseDate = version.ReleaseDate
	}
	sb.WriteString(fmt.Sprintf("<p class=\"meta\"><strong>Data di rilascio</strong>: %s</p>", releaseDate))

	epics := hierarchy.Epics
	epicChildren := hierarchy.EpicChildren
	standaloneIssues := hierarchy.StandaloneIssues
	subtaskMap := hierarchy.SubtaskMap

	printedSubtasks := make(map[string]bool)

	if len(epics) > 0 {
		sb.WriteString("<h2>🎯 Epic</h2>")
		for _, epic := range epics {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, epic.Key)
			sb.WriteString("<div class=\"epic-box\">")
			sb.WriteString(fmt.Sprintf("<h3><a href=\"%s\">%s</a> %s</h3>", issueURL, epic.Key, epic.Fields.Summary))

			if children, ok := epicChildren[epic.Key]; ok && len(children) > 0 {
				sb.WriteString("<ul>")
				for _, child := range children {
					childURL := fmt.Sprintf("%s/browse/%s", baseURL, child.Key)
					sb.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a>: %s", childURL, child.Key, child.Fields.Summary))

					if includeSubtasks {
						if subtasks, ok := subtaskMap[child.Key]; ok && len(subtasks) > 0 {
							sb.WriteString("<ul>")
							for _, subtask := range subtasks {
								subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
								sb.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a>: %s</li>", subtaskURL, subtask.Key, subtask.Fields.Summary))
								printedSubtasks[subtask.Key] = true
							}
							sb.WriteString("</ul>")
						}
					}
					sb.WriteString("</li>")
				}
				sb.WriteString("</ul>")
			}
			sb.WriteString("</div>")
		}
	}

	typeEmoji := map[string]string{
		"Story":       "✨",
		"Task":        "📝",
		"Improvement": "🔧",
		"Bug":         "🐛",
	}

	for issueType, issuesList := range standaloneIssues {
		if len(issuesList) == 0 {
			continue
		}

		emoji := typeEmoji[issueType]
		if emoji == "" {
			emoji = "•"
		}

		sb.WriteString(fmt.Sprintf("<h2>%s %s</h2><ul>", emoji, issueType))

		for _, issue := range issuesList {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, issue.Key)
			sb.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a>: %s", issueURL, issue.Key, issue.Fields.Summary))

			if includeSubtasks {
				if subtasks, ok := subtaskMap[issue.Key]; ok && len(subtasks) > 0 {
					sb.WriteString("<ul>")
					for _, subtask := range subtasks {
						subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
						sb.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a>: %s</li>", subtaskURL, subtask.Key, subtask.Fields.Summary))
						printedSubtasks[subtask.Key] = true
					}
					sb.WriteString("</ul>")
				}
			}
			sb.WriteString("</li>")
		}
		sb.WriteString("</ul>")
	}

	var orphanedSubtasks []jira.Issue
	if includeSubtasks {
		for _, subtasks := range subtaskMap {
			for _, subtask := range subtasks {
				if _, printed := printedSubtasks[subtask.Key]; !printed {
					orphanedSubtasks = append(orphanedSubtasks, subtask)
				}
			}
		}
	}

	if len(orphanedSubtasks) > 0 {
		sb.WriteString("<h2>📎 Sub-task Aggiuntivi</h2>")
		sb.WriteString("<p><em>(Ticket con fixVersion, ma genitore non in questa release o completato)</em></p><ul>")
		for _, subtask := range orphanedSubtasks {
			subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
			sb.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a>: %s</li>", subtaskURL, subtask.Key, subtask.Fields.Summary))
		}
		sb.WriteString("</ul>")
	}

	sb.WriteString("</body></html>")

	return sb.String()
}

func init() {
	rootCmd.AddCommand(changelogCmd)
	changelogCmd.Flags().StringP("project", "p", "", "Chiave del progetto Jira (es. PROJ)")
	changelogCmd.Flags().StringP("output", "o", "", "File di output per salvare il changelog")
	changelogCmd.Flags().StringP("format", "f", "markdown", "Formato del changelog: markdown, slack, html")
	changelogCmd.Flags().BoolP("include-subtasks", "s", false, "Includi i sub-task nel changelog")
}
