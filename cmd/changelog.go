package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"jira-release-manager/internal/jira"

	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Genera un changelog in formato Markdown per la prossima release.",
	Long: `Genera un changelog formattato in Markdown basato sui ticket della prossima release.
Il changelog raggruppa i ticket per tipo e può essere salvato su file.`,
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

		nextVersion, err := jira.FindNextReleaseVersion(client, projectKey)
		if err != nil {
			log.Fatalf("Errore: %v", err)
		}

		issues, err := jira.GetIssuesForVersion(client, projectKey, nextVersion.Name)
		if err != nil {
			log.Fatalf("Errore: %v", err)
		}

		var changelog string
		switch format {
		case "markdown", "md":
			changelog = generateMarkdownChangelog(nextVersion, issues, includeSubtasks, client.BaseURL)
		case "slack":
			changelog = generateSlackChangelog(nextVersion, issues, includeSubtasks, client.BaseURL)
		case "html":
			changelog = generateHTMLChangelog(nextVersion, issues, includeSubtasks, client.BaseURL)
		default:
			changelog = generateMarkdownChangelog(nextVersion, issues, includeSubtasks, client.BaseURL)
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

func generateMarkdownChangelog(version *jira.Version, issues []jira.Issue, includeSubtasks bool, baseURL string) string {
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

	// Raggruppa per tipo
	groupedIssues := make(map[string][]jira.Issue)
	subtaskMap := make(map[string][]jira.Issue)

	for _, issue := range issues {
		if issue.Fields.IssueType.Subtask {
			if includeSubtasks && issue.Fields.Parent != nil {
				parentKey := issue.Fields.Parent.Key
				subtaskMap[parentKey] = append(subtaskMap[parentKey], issue)
			}
		} else {
			issueType := issue.Fields.IssueType.Name
			groupedIssues[issueType] = append(groupedIssues[issueType], issue)
		}
	}

	// Ordine preferito dei tipi
	preferredOrder := []string{"Epic", "Story", "Task", "Improvement", "Bug", "Sub-task"}
	typeEmoji := map[string]string{
		"Epic":        "🎯",
		"Story":       "✨",
		"Task":        "📝",
		"Improvement": "🔧",
		"Bug":         "🐛",
		"Sub-task":    "📌",
	}

	for _, issueType := range preferredOrder {
		issuesList, ok := groupedIssues[issueType]
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

			// Aggiungi sub-task se richiesto
			if includeSubtasks {
				if subtasks, hasSubtasks := subtaskMap[issue.Key]; hasSubtasks {
					for _, subtask := range subtasks {
						subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
						sb.WriteString(fmt.Sprintf("  - [%s](%s): %s\n", subtask.Key, subtaskURL, subtask.Fields.Summary))
					}
				}
			}
		}
		sb.WriteString("\n")
	}

	// Aggiungi eventuali tipi non previsti
	for issueType, issuesList := range groupedIssues {
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

	return sb.String()
}

func generateSlackChangelog(version *jira.Version, issues []jira.Issue, includeSubtasks bool, baseURL string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*📋 Changelog - Versione %s*\n\n", version.Name))

	releaseDate := time.Now().Format("2006-01-02")
	if version.ReleaseDate != "" {
		releaseDate = version.ReleaseDate
	}
	sb.WriteString(fmt.Sprintf("*Data di rilascio*: %s\n\n", releaseDate))

	groupedIssues := make(map[string][]jira.Issue)
	for _, issue := range issues {
		if !issue.Fields.IssueType.Subtask {
			issueType := issue.Fields.IssueType.Name
			groupedIssues[issueType] = append(groupedIssues[issueType], issue)
		}
	}

	typeEmoji := map[string]string{
		"Story":       ":sparkles:",
		"Task":        ":memo:",
		"Improvement": ":wrench:",
		"Bug":         ":bug:",
	}

	for issueType, issuesList := range groupedIssues {
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

func generateHTMLChangelog(version *jira.Version, issues []jira.Issue, includeSubtasks bool, baseURL string) string {
	var sb strings.Builder

	sb.WriteString("<html><head><meta charset=\"UTF-8\"><title>Changelog</title>")
	sb.WriteString("<style>body{font-family:Arial,sans-serif;max-width:800px;margin:40px auto;padding:20px;}")
	sb.WriteString("h1{color:#0052CC;}h2{color:#333;border-bottom:2px solid #0052CC;padding-bottom:5px;}")
	sb.WriteString("ul{list-style-type:none;padding-left:0;}li{margin:10px 0;}a{color:#0052CC;text-decoration:none;}")
	sb.WriteString("a:hover{text-decoration:underline;}.meta{color:#666;font-size:14px;}</style></head><body>")

	sb.WriteString(fmt.Sprintf("<h1>📋 Changelog - Versione %s</h1>", version.Name))

	releaseDate := time.Now().Format("2006-01-02")
	if version.ReleaseDate != "" {
		releaseDate = version.ReleaseDate
	}
	sb.WriteString(fmt.Sprintf("<p class=\"meta\"><strong>Data di rilascio</strong>: %s</p>", releaseDate))

	groupedIssues := make(map[string][]jira.Issue)
	for _, issue := range issues {
		if !issue.Fields.IssueType.Subtask {
			issueType := issue.Fields.IssueType.Name
			groupedIssues[issueType] = append(groupedIssues[issueType], issue)
		}
	}

	typeEmoji := map[string]string{
		"Story":       "✨",
		"Task":        "📝",
		"Improvement": "🔧",
		"Bug":         "🐛",
	}

	for issueType, issuesList := range groupedIssues {
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
			sb.WriteString(fmt.Sprintf("<li><strong><a href=\"%s\">%s</a></strong>: %s</li>",
				issueURL, issue.Key, issue.Fields.Summary))
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
