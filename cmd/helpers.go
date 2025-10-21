package cmd

import (
	"fmt"
	"log"
	"os"

	"jira-release-manager/internal/jira"

	"github.com/AlecAivazis/survey/v2"
)

// selectJiraVersion mostra un prompt interattivo per selezionare una versione.
// Ritorna la versione selezionata dall'utente.
func selectJiraVersion(client *jira.Client, projectKey string) *jira.Version {
	fmt.Printf("🔎 Ricerca versioni per il progetto %s...\n", projectKey)
	versions, err := jira.GetAllProjectVersions(client, projectKey)
	if err != nil {
		log.Fatalf("Errore nel recupero delle versioni: %v", err)
	}

	if len(versions) == 0 {
		log.Fatalf("Nessuna versione trovata per il progetto %s", projectKey)
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

	// survey.WithStdio è un fix per funzionare correttamente negli IDE
	err = survey.AskOne(prompt, &selectedOption, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	if err != nil {
		log.Fatalf("Selezione annullata o fallita: %v", err)
	}

	// Restituisci la versione completa
	selectedVersion := optionMap[selectedOption]
	fmt.Println() // Aggiunge una riga vuota dopo la selezione
	return &selectedVersion
}
