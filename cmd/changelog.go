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
Il changelog raggruppa i ticket per tipo e può essere salvato su file.
Usa --version per specificare una versione esatta.`,
	Example: `  jira-release-manager changelog --project PROJ
  jira-release-manager changelog -p PROJ --output CHANGELOG.md
  jira-release-manager changelog -p PROJ --format slack
  jira-release-manager changelog -p PROJ -v "Release 1.2.3"`,
	Run: func(cmd *cobra.Command, args []string) {
		projectKey, _ := cmd.Flags().GetString("project")
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		includeSubtasks, _ := cmd.Flags().GetBool("include-subtasks")
		versionName, _ := cmd.Flags().GetString("version")

		if projectKey == "" {
			log.Fatal("Il flag --project è obbligatorio.")
		}

		client, err := jira.NewClient()
		if err != nil {
			log.Fatalf("Errore: %v", err)
		}

		var versionToFetch *jira.Version

		// Se la versione non è specificata, trova la prossima
		if versionName == "" {
			fmt.Printf("🔎 Ricerca della prossima release per il progetto %s...\n", projectKey)
			nextVersion, err := jira.FindNextReleaseVersion(client, projectKey)
			if err != nil {
				log.Fatalf("Errore: %v", err)
			}
			versionToFetch = nextVersion
		} else {
			// Se la versione è specificata, cerca quella
			fmt.Printf("🔎 Ricerca della versione specificata '%s'...\n", versionName)
			allVersions, err := jira.GetAllProjectVersions(client, projectKey)
			if err != nil {
				log.Fatalf("Errore nel recupero versioni: %v", err)
			}

			var found *jira.Version
			for i, v := range allVersions {
				if v.Name == versionName {
					found = &allVersions[i]
					break
				}
			}
			if found == nil {
				log.Fatalf("Errore: Versione '%s' non trovata per il progetto %s", versionName, projectKey)
			}
			versionToFetch = found
		}

		fmt.Printf("✅ Utilizzo della versione: %s\n", versionToFetch.Name)

		issues, err := jira.GetIssuesForVersion(client, projectKey, versionToFetch.Name)
		if err != nil {
			log.Fatalf("Errore nel recupero dei ticket: %v", err)
		}

		var changelog string
		switch format {
		case "markdown", "md":
			changelog = generateMarkdownChangelog(versionToFetch, issues, includeSubtasks, client.BaseURL)
		case "slack":
			changelog = generateSlackChangelog(versionToFetch, issues, includeSubtasks, client.BaseURL)
		case "html":
			changelog = generateHTMLChangelog(versionToFetch, issues, includeSubtasks, client.BaseURL)
		default:
			changelog = generateMarkdownChangelog(versionToFetch, issues, includeSubtasks, client.BaseURL)
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

	// Organizza i ticket usando la stessa logica di next-release
	epics := make(map[string]jira.Issue)
	epicChildren := make(map[string][]jira.Issue)
	standaloneIssues := make(map[string][]jira.Issue)
	subtaskMap := make(map[string][]jira.Issue)

	// Identifica epics e subtask
	for _, issue := range issues {
		issueType := strings.ToLower(issue.Fields.IssueType.Name)

		if issue.Fields.IssueType.Subtask {
			if includeSubtasks && issue.Fields.Parent != nil {
				parentKey := issue.Fields.Parent.Key
				subtaskMap[parentKey] = append(subtaskMap[parentKey], issue)
			}
		} else if issueType == "epic" {
			epics[issue.Key] = issue
		}
	}

	// Organizza children con la stessa logica (epic e parent)
	for _, issue := range issues {
		if issue.Fields.IssueType.Subtask || strings.ToLower(issue.Fields.IssueType.Name) == "epic" {
			continue
		}

		epicKey := ""

		// Priorità 1: campo Epic
		if issue.Fields.Epic != nil && issue.Fields.Epic.Key != "" {
			epicKey = issue.Fields.Epic.Key
		}

		// Priorità 2: campo Parent se punta a un Epic
		if epicKey == "" && issue.Fields.Parent != nil && issue.Fields.Parent.Key != "" {
			parentKey := issue.Fields.Parent.Key
			if _, isEpic := epics[parentKey]; isEpic {
				epicKey = parentKey
			}
		}

		if epicKey != "" && epics[epicKey].Key != "" {
			epicChildren[epicKey] = append(epicChildren[epicKey], issue)
			continue
		}

		standaloneIssues[issue.Fields.IssueType.Name] = append(standaloneIssues[issue.Fields.IssueType.Name], issue)
	}

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

	// Usa la stessa logica
	epics := make(map[string]jira.Issue)
	epicChildren := make(map[string][]jira.Issue)
	standaloneIssues := make(map[string][]jira.Issue)

	for _, issue := range issues {
		if issue.Fields.IssueType.Subtask {
			continue
		}
		issueType := strings.ToLower(issue.Fields.IssueType.Name)
		if issueType == "epic" {
			epics[issue.Key] = issue
		}
	}

	for _, issue := range issues {
		if issue.Fields.IssueType.Subtask || strings.ToLower(issue.Fields.IssueType.Name) == "epic" {
			continue
		}

		epicKey := ""
		if issue.Fields.Epic != nil && issue.Fields.Epic.Key != "" {
			epicKey = issue.Fields.Epic.Key
		}
		if epicKey == "" && issue.Fields.Parent != nil && issue.Fields.Parent.Key != "" {
			if _, isEpic := epics[issue.Fields.Parent.Key]; isEpic {
				epicKey = issue.Fields.Parent.Key
			}
		}

		if epicKey != "" && epics[epicKey].Key != "" {
			epicChildren[epicKey] = append(epicChildren[epicKey], issue)
			continue
		}

		standaloneIssues[issue.Fields.IssueType.Name] = append(standaloneIssues[issue.Fields.IssueType.Name], issue)
	}

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

func generateHTMLChangelog(version *jira.Version, issues []jira.Issue, includeSubtasks bool, baseURL string) string {
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

	// Usa la stessa logica
	epics := make(map[string]jira.Issue)
	epicChildren := make(map[string][]jira.Issue)
	standaloneIssues := make(map[string][]jira.Issue)
	subtaskMap := make(map[string][]jira.Issue)

	for _, issue := range issues {
		issueType := strings.ToLower(issue.Fields.IssueType.Name)
		if issue.Fields.IssueType.Subtask {
			if includeSubtasks && issue.Fields.Parent != nil {
				subtaskMap[issue.Fields.Parent.Key] = append(subtaskMap[issue.Fields.Parent.Key], issue)
			}
		} else if issueType == "epic" {
			epics[issue.Key] = issue
		}
	}

	for _, issue := range issues {
		if issue.Fields.IssueType.Subtask || strings.ToLower(issue.Fields.IssueType.Name) == "epic" {
			continue
		}

		epicKey := ""
		if issue.Fields.Epic != nil && issue.Fields.Epic.Key != "" {
			epicKey = issue.Fields.Epic.Key
		}
		if epicKey == "" && issue.Fields.Parent != nil && issue.Fields.Parent.Key != "" {
			if _, isEpic := epics[issue.Fields.Parent.Key]; isEpic {
				epicKey = issue.Fields.Parent.Key
			}
		}

		if epicKey != "" && epics[epicKey].Key != "" {
			epicChildren[epicKey] = append(epicChildren[epicKey], issue)
			continue
		}

		standaloneIssues[issue.Fields.IssueType.Name] = append(standaloneIssues[issue.Fields.IssueType.Name], issue)
	}

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
					}
					sb.WriteString("</ul>")
				}
			}
			sb.WriteString("</li>")
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
	changelogCmd.Flags().StringP("version", "v", "", "Nome esatto della versione Jira (opzionale, default: prossima release)") // <-- Aggiunto
}
