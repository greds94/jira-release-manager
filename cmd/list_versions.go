package cmd

import (
	"fmt"
	"jira-release-manager/internal/jira"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listVersionsCmd = &cobra.Command{
	Use:   "list-versions",
	Short: "Mostra una tabella di tutte le versioni per un progetto.",
	Long: `Recupera tutte le versioni (rilasciate, non rilasciate, archiviate) 
per un progetto e le mostra in una tabella.`,
	Example: `  jira-release-manager list-versions -p PROJ`,

	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🔎 Ricerca versioni per il progetto %s...\n\n", projectKey)
		versions, err := jira.GetAllProjectVersions(jiraClient, projectKey)
		if err != nil {
			return err
		}

		if len(versions) == 0 {
			fmt.Println("Nessuna versione trovata per questo progetto.")
			return nil
		}

		// Inizializza tabwriter
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 2, ' ', 0)

		fmt.Fprintln(w, "NOME VERSIONE\tSTATO\tDATA RILASCIO\tDATA INIZIO\tDESCRIZIONE")
		fmt.Fprintln(w, "---------------\t-----\t-------------\t-------------\t-----------")

		for _, v := range versions {
			status := "Non Rilasciata"
			if v.Released {
				status = "Rilasciata"
			}
			if v.Archived {
				status = "Archiviata"
			}

			releaseDate := v.ReleaseDate
			if releaseDate == "" {
				releaseDate = "N/D"
			}

			startDate := v.StartDate
			if startDate == "" {
				startDate = "N/D"
			}

			desc := v.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", v.Name, status, releaseDate, startDate, desc)
		}

		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listVersionsCmd)
}
