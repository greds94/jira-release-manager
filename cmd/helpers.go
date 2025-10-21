package cmd

import (
	"fmt"
	"os"

	"jira-release-manager/internal/jira"

	"github.com/AlecAivazis/survey/v2"
)

// selectJiraVersion mostra un prompt interattivo per selezionare una versione.
func selectJiraVersion(client *jira.Client, projectKey string) (*jira.Version, error) {
	fmt.Printf("🔎 Ricerca versioni per il progetto %s...\n", projectKey)
	versions, err := jira.GetAllProjectVersions(client, projectKey)
	if err != nil {
		return nil, fmt.Errorf("errore nel recupero delle versioni: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("nessuna versione trovata per il progetto %s", projectKey)
	}

	// Prepara le opzioni per il selettore
	var options []string
	optionMap := make(map[string]jira.Version) // Mappa per recuperare la versione scelta

	for _, v := range versions {
		status := "Non Rilasciata"
		if v.Released {
			status = "Rilasciata"
		}
		if v.Archived {
			status = "Archiviata"
		}

		date := v.ReleaseDate
		if date == "" {
			date = "N/D"
		}

		// Formatta la stringa per l'opzione
		optionStr := fmt.Sprintf("%s (%s, Data: %s)", v.Name, status, date)
		options = append(options, optionStr)
		optionMap[optionStr] = v
	}

	// Crea il prompt
	var selectedOption string
	prompt := &survey.Select{
		Message:  "Seleziona una versione:",
		Options:  options,
		PageSize: 15, // Mostra 15 opzioni alla volta
	}

	err = survey.AskOne(prompt, &selectedOption, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if err != nil {
		return nil, fmt.Errorf("selezione annullata o fallita: %w", err)
	}

	selectedVersion := optionMap[selectedOption]
	fmt.Println()
	return &selectedVersion, nil
}
