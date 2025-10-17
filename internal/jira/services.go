package jira

import (
	"fmt"
	"net/url"
	"sort"
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
	// JQL per trovare tutte le issue padre nella versione specificata
	jql := fmt.Sprintf(`project = "%s" AND fixVersion = "%s" AND issuetype not in (Sub-task, Sub-bug)`, projectKey, versionName)

	// Costruisci l'URL con query parameters secondo la documentazione
	params := url.Values{}
	params.Add("jql", jql)
	params.Add("startAt", "0")
	params.Add("maxResults", "100")
	params.Add("fields", "summary,status,assignee,priority,issuetype,parent,subtasks")

	// Usa l'endpoint GET /rest/api/3/search/jql
	endpoint := fmt.Sprintf("/rest/api/3/search/jql?%s", params.Encode())

	fmt.Printf("DEBUG: Sending GET to %s\n", endpoint)

	var searchResults SearchResults
	if err := client.GetJSON(endpoint, &searchResults); err != nil {
		return nil, fmt.Errorf("errore nella ricerca JQL: %w", err)
	}

	var allIssues []Issue
	allIssues = append(allIssues, searchResults.Issues...)

	// Per ogni issue padre, recupera i sub-task
	for _, issue := range searchResults.Issues {
		if len(issue.Fields.Subtasks) > 0 {
			for _, subtaskRef := range issue.Fields.Subtasks {
				subtask, err := GetIssue(client, subtaskRef.Key)
				if err != nil {
					fmt.Printf("Attenzione: impossibile recuperare il sub-task %s: %v\n", subtaskRef.Key, err)
					continue
				}
				allIssues = append(allIssues, *subtask)
			}
		}
	}

	return allIssues, nil
}

// GetIssue recupera un singolo ticket tramite la sua chiave
func GetIssue(client *Client, issueKey string) (*Issue, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

	var issue Issue
	if err := client.GetJSON(endpoint, &issue); err != nil {
		return nil, fmt.Errorf("impossibile recuperare il ticket %s: %w", issueKey, err)
	}

	return &issue, nil
}
