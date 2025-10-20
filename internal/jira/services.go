package jira

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// FindNextReleaseVersion trova la prima versione non rilasciata per un dato progetto.
func FindNextReleaseVersion(client *Client, projectKey string) (*Version, error) {
	// Recupera il progetto con le versioni usando API v3
	endpoint := fmt.Sprintf("/rest/api/3/project/%s?expand=versions", projectKey)

	var project Project
	if err := client.GetJSON(endpoint, &project); err != nil {
		return nil, fmt.Errorf("impossibile recuperare il progetto %s: %w", projectKey, err)
	}

	// Filtra le versioni non rilasciate e non archiviate
	var unreleasedVersions []Version
	for _, v := range project.Versions {
		if !v.Released && !v.Archived {
			unreleasedVersions = append(unreleasedVersions, v)
		}
	}

	if len(unreleasedVersions) == 0 {
		return nil, fmt.Errorf("nessuna versione non rilasciata trovata per il progetto %s", projectKey)
	}

	// Ordina le versioni per data di rilascio (se disponibile) o per nome
	sort.Slice(unreleasedVersions, func(i, j int) bool {
		// Se entrambe hanno date di rilascio, usa quelle
		if unreleasedVersions[i].ReleaseDate != "" && unreleasedVersions[j].ReleaseDate != "" {
			return unreleasedVersions[i].ReleaseDate < unreleasedVersions[j].ReleaseDate
		}
		// Altrimenti ordina per nome
		return unreleasedVersions[i].Name < unreleasedVersions[j].Name
	})

	return &unreleasedVersions[0], nil
}

// GetIssuesForVersion recupera tutti i ticket per una versione specifica usando /rest/api/3/search/jql
func GetIssuesForVersion(client *Client, projectKey string, versionName string) ([]Issue, error) {
	fmt.Print("⏳ Recupero ticket in rilascio...")

	// JQL per trovare tutte le issue nella versione specificata, escludendo quelle completate
	jql := fmt.Sprintf(`project = "%s" AND fixVersion = "%s" AND statusCategory != Done AND issuetype not in (Sub-task, Sub-bug)`, projectKey, versionName)

	params := url.Values{}
	params.Add("jql", jql)
	params.Add("startAt", "0")
	params.Add("maxResults", "100")
	params.Add("fields", "summary,status,assignee,priority,issuetype,parent,subtasks,epic,labels")

	endpoint := fmt.Sprintf("/rest/api/3/search/jql?%s", params.Encode())

	var searchResults SearchResults
	if err := client.GetJSON(endpoint, &searchResults); err != nil {
		fmt.Println(" ❌")
		return nil, fmt.Errorf("errore nella ricerca JQL: %w", err)
	}
	fmt.Print(" ✓\n")

	var allIssues []Issue
	issueMap := make(map[string]*Issue)
	epicKeysInRelease := make(map[string]bool) // Epic che sono direttamente nella release

	// Prima passata: identifica gli Epic nella release e aggiungi tutte le issue
	for _, issue := range searchResults.Issues {
		issueCopy := issue
		allIssues = append(allIssues, issueCopy)
		issueMap[issue.Key] = &issueCopy

		if strings.ToLower(issue.Fields.IssueType.Name) == "epic" {
			epicKeysInRelease[issue.Key] = true
		}
	}

	// Recupera i sub-task per le issue nella release
	fmt.Print("⏳ Recupero sub-task...")
	subtaskCount := 0
	for _, issue := range searchResults.Issues {
		if len(issue.Fields.Subtasks) > 0 {
			for _, subtaskRef := range issue.Fields.Subtasks {
				if _, exists := issueMap[subtaskRef.Key]; exists {
					continue // già presente
				}

				subtask, err := GetIssue(client, subtaskRef.Key)
				if err != nil {
					continue
				}

				if subtask.IsCompleted() {
					continue
				}

				allIssues = append(allIssues, *subtask)
				issueMap[subtask.Key] = subtask
				subtaskCount++
			}
		}
	}
	fmt.Printf(" ✓ (%d trovati)\n", subtaskCount)

	// Recupera le Story/Task che appartengono agli Epic nella release
	if len(epicKeysInRelease) > 0 {
		fmt.Print("⏳ Recupero story collegate agli epic...")

		epicKeys := make([]string, 0, len(epicKeysInRelease))
		for key := range epicKeysInRelease {
			epicKeys = append(epicKeys, key)
		}

		epicJQL := fmt.Sprintf(`project = "%s" AND statusCategory != Done AND "Epic Link" in (%s)`, projectKey, strings.Join(wrapKeys(epicKeys), ","))

		params := url.Values{}
		params.Add("jql", epicJQL)
		params.Add("startAt", "0")
		params.Add("maxResults", "100")
		params.Add("fields", "summary,status,assignee,priority,issuetype,parent,subtasks,epic,labels")

		epicEndpoint := fmt.Sprintf("/rest/api/3/search/jql?%s", params.Encode())

		var epicResults SearchResults
		storyCount := 0
		if err := client.GetJSON(epicEndpoint, &epicResults); err == nil {
			for _, story := range epicResults.Issues {
				if _, exists := issueMap[story.Key]; !exists {
					storyCopy := story
					allIssues = append(allIssues, storyCopy)
					issueMap[story.Key] = &storyCopy
					storyCount++

					// Recupera i sub-task della story
					if len(story.Fields.Subtasks) > 0 {
						for _, subtaskRef := range story.Fields.Subtasks {
							if _, exists := issueMap[subtaskRef.Key]; exists {
								continue
							}

							subtask, err := GetIssue(client, subtaskRef.Key)
							if err != nil {
								continue
							}

							if subtask.IsCompleted() {
								continue
							}

							allIssues = append(allIssues, *subtask)
							issueMap[subtask.Key] = subtask
						}
					}
				}
			}
		}
		fmt.Printf(" ✓ (%d trovate)\n", storyCount)
	}

	fmt.Println()
	return allIssues, nil
}

// wrapKeys avvolge le chiavi con virgolette per la JQL
func wrapKeys(keys []string) []string {
	wrapped := make([]string, len(keys))
	for i, key := range keys {
		wrapped[i] = fmt.Sprintf(`"%s"`, key)
	}
	return wrapped
}

// GetIssue recupera un singolo ticket tramite la sua chiave
func GetIssue(client *Client, issueKey string) (*Issue, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary,status,assignee,priority,labels", issueKey)

	var issue Issue
	if err := client.GetJSON(endpoint, &issue); err != nil {
		return nil, fmt.Errorf("impossibile recuperare il ticket %s: %w", issueKey, err)
	}

	return &issue, nil
}
