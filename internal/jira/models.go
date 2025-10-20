package jira

// Project rappresenta un progetto Jira
type Project struct {
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	Versions []Version `json:"versions"`
}

// Version rappresenta una versione di rilascio
type Version struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
	Released    bool   `json:"released"`
	ReleaseDate string `json:"releaseDate"`
	StartDate   string `json:"startDate"`
}

// SearchResults rappresenta i risultati di una ricerca JQL
type SearchResults struct {
	Issues     []Issue `json:"issues"`
	Total      int     `json:"total"`
	MaxResults int     `json:"maxResults"`
	StartAt    int     `json:"startAt"`
}

// Issue rappresenta un ticket Jira
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contiene i campi di un ticket
type IssueFields struct {
	Summary     string      `json:"summary"`
	Description interface{} `json:"description"` // Può essere string o object (ADF format)
	Status      Status      `json:"status"`
	Priority    *Priority   `json:"priority"`
	IssueType   IssueType   `json:"issuetype"`
	Assignee    *User       `json:"assignee"`
	Reporter    *User       `json:"reporter"`
	Parent      *IssueRef   `json:"parent"`
	Epic        *EpicLink   `json:"epic,omitempty"` // Link all'epic
	Subtasks    []IssueRef  `json:"subtasks"`
	Labels      []string    `json:"labels,omitempty"` // <<< CAMPO AGGIUNTO
}

// Status rappresenta lo stato di un ticket
type Status struct {
	Name           string         `json:"name"`
	ID             string         `json:"id"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

// StatusCategory rappresenta la categoria di stato (To Do, In Progress, Done)
type StatusCategory struct {
	Key  string `json:"key"`  // "new", "indeterminate", "done"
	Name string `json:"name"` // "To Do", "In Progress", "Done"
}

// Priority rappresenta la priorità
type Priority struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// IssueType rappresenta il tipo di ticket
type IssueType struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Subtask bool   `json:"subtask"`
}

// User rappresenta un utente Jira
type User struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

// IssueRef rappresenta un riferimento a un ticket (per parent/subtasks)
type IssueRef struct {
	ID     string       `json:"id"`
	Key    string       `json:"key"`
	Fields *IssueFields `json:"fields,omitempty"`
}

// EpicLink rappresenta il collegamento a un Epic
type EpicLink struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
}

// GetDescriptionText estrae il testo dalla description (gestisce sia string che ADF format)
func (i *Issue) GetDescriptionText() string {
	if i.Fields.Description == nil {
		return ""
	}

	// Se è già una stringa, restituiscila
	if str, ok := i.Fields.Description.(string); ok {
		return str
	}

	// Se è un oggetto ADF (Atlassian Document Format), prova a estrarre il testo
	if descMap, ok := i.Fields.Description.(map[string]interface{}); ok {
		// Estrai il testo dai content nodes (semplificato)
		if content, ok := descMap["content"].([]interface{}); ok && len(content) > 0 {
			var text string
			for _, node := range content {
				if nodeMap, ok := node.(map[string]interface{}); ok {
					if nodeContent, ok := nodeMap["content"].([]interface{}); ok {
						for _, textNode := range nodeContent {
							if textMap, ok := textNode.(map[string]interface{}); ok {
								if t, ok := textMap["text"].(string); ok {
									text += t + " "
								}
							}
						}
					}
				}
			}
			return text
		}
	}

	return ""
}

// IsCompleted verifica se il ticket è nello stato completato
func (i *Issue) IsCompleted() bool {
	return i.Fields.Status.StatusCategory.Key == "done"
}
