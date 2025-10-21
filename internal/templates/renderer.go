package templates

import (
	"fmt"
	"strings"
	"time"

	"jira-release-manager/internal/jira"
	"jira-release-manager/internal/organizer"
)

// RenderMarkdown genera un changelog in formato Markdown
func RenderMarkdown(version *jira.Version, hierarchy *organizer.ReleaseHierarchy, includeSubtasks bool, baseURL string) string {
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

	epics := hierarchy.Epics
	epicChildren := hierarchy.EpicChildren
	standaloneIssues := hierarchy.StandaloneIssues
	subtaskMap := hierarchy.SubtaskMap
	printedSubtasks := make(map[string]bool)

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

// RenderTeams genera un changelog in formato Markdown per Microsoft Teams
func RenderTeams(version *jira.Version, hierarchy *organizer.ReleaseHierarchy, includeSubtasks bool, baseURL string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**📋 Changelog - Versione %s**\n\n", version.Name))

	releaseDate := time.Now().Format("2006-01-02")
	if version.ReleaseDate != "" {
		releaseDate = version.ReleaseDate
	}
	sb.WriteString(fmt.Sprintf("**Data di rilascio**: %s\n\n", releaseDate))

	if version.Description != "" {
		sb.WriteString(fmt.Sprintf("**Descrizione**: %s\n\n", version.Description))
	}

	sb.WriteString("---\n\n")

	epics := hierarchy.Epics
	epicChildren := hierarchy.EpicChildren
	standaloneIssues := hierarchy.StandaloneIssues
	subtaskMap := hierarchy.SubtaskMap
	printedSubtasks := make(map[string]bool)

	if len(epics) > 0 {
		sb.WriteString("**🎯 Epic**\n\n")
		for _, epic := range epics {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, epic.Key)
			// Titolo Epic in grassetto
			sb.WriteString(fmt.Sprintf("**[%s](%s)** %s\n\n", epic.Key, issueURL, epic.Fields.Summary))

			if children, ok := epicChildren[epic.Key]; ok && len(children) > 0 {
				for _, child := range children {
					childURL := fmt.Sprintf("%s/browse/%s", baseURL, child.Key)
					// Elenco puntato per i figli
					sb.WriteString(fmt.Sprintf("* **[%s](%s)**: %s\n", child.Key, childURL, child.Fields.Summary))

					if includeSubtasks {
						if subtasks, ok := subtaskMap[child.Key]; ok {
							for _, subtask := range subtasks {
								subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
								// Sotto-elenco per subtask
								sb.WriteString(fmt.Sprintf("  * [%s](%s): %s\n", subtask.Key, subtaskURL, subtask.Fields.Summary))
								printedSubtasks[subtask.Key] = true
							}
						}
					}
				}
				sb.WriteString("\n")
			}
		}
	}

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

		sb.WriteString(fmt.Sprintf("**%s %s**\n\n", emoji, issueType))

		for _, issue := range issuesList {
			issueURL := fmt.Sprintf("%s/browse/%s", baseURL, issue.Key)
			sb.WriteString(fmt.Sprintf("* **[%s](%s)**: %s\n", issue.Key, issueURL, issue.Fields.Summary))

			if includeSubtasks {
				if subtasks, ok := subtaskMap[issue.Key]; ok {
					for _, subtask := range subtasks {
						subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
						sb.WriteString(fmt.Sprintf("  * [%s](%s): %s\n", subtask.Key, subtaskURL, subtask.Fields.Summary))
						printedSubtasks[subtask.Key] = true
					}
				}
			}
		}
		sb.WriteString("\n")
	}

	for issueType, issuesList := range standaloneIssues {
		found := false
		for _, preferred := range preferredOrder {
			if issueType == preferred {
				found = true
				break
			}
		}
		if !found && len(issuesList) > 0 {
			sb.WriteString(fmt.Sprintf("**• %s**\n\n", issueType))
			for _, issue := range issuesList {
				issueURL := fmt.Sprintf("%s/browse/%s", baseURL, issue.Key)
				sb.WriteString(fmt.Sprintf("* **[%s](%s)**: %s\n", issue.Key, issueURL, issue.Fields.Summary))
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
		sb.WriteString("**📎 Sub-task Aggiuntivi**\n\n")
		sb.WriteString("*(Ticket con fixVersion, ma genitore non in questa release o completato)*\n\n")
		for _, subtask := range orphanedSubtasks {
			subtaskURL := fmt.Sprintf("%s/browse/%s", baseURL, subtask.Key)
			sb.WriteString(fmt.Sprintf("* [%s](%s): %s\n", subtask.Key, subtaskURL, subtask.Fields.Summary))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
