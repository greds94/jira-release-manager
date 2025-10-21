package cmd

import (
	"fmt"
	"jira-release-manager/internal/jira"
	"os"

	"jira-release-manager/internal/organizer"
	"jira-release-manager/internal/templates"

	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Genera un changelog in formato Markdown per una versione.",
	Long: `Permette di selezionare interattivamente una versione e genera un changelog 
formattato in Markdown (o altri formati) basato sui ticket.`,
	Example: `  jira-release-manager changelog -p PROJ
  jira-release-manager changelog -p PROJ --output CHANGELOG.md
  jira-release-manager changelog -p PROJ --format teams`,

	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		includeSubtasks, _ := cmd.Flags().GetBool("include-subtasks")

		versionToFetch, err := selectJiraVersion(jiraClient, projectKey)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Generazione changelog per la versione: %s\n", versionToFetch.Name)

		issues, err := jira.GetIssuesForVersion(jiraClient, projectKey, versionToFetch.Name)
		if err != nil {
			return fmt.Errorf("errore nel recupero dei ticket: %w", err)
		}

		hierarchy := organizer.NewReleaseHierarchy(issues, false)

		var changelog string
		switch format {
		case "markdown", "md":
			changelog = templates.RenderMarkdown(versionToFetch, hierarchy, includeSubtasks, jiraClient.BaseURL)
		case "teams":
			changelog = templates.RenderTeams(versionToFetch, hierarchy, includeSubtasks, jiraClient.BaseURL)
		default:
			changelog = templates.RenderMarkdown(versionToFetch, hierarchy, includeSubtasks, jiraClient.BaseURL)
		}

		if outputFile != "" {
			err := os.WriteFile(outputFile, []byte(changelog), 0644)
			if err != nil {
				return fmt.Errorf("errore nel salvataggio del file: %w", err)
			}
			fmt.Printf("✅ Changelog salvato in: %s\n", outputFile)
		} else {
			fmt.Println(changelog)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(changelogCmd)
	changelogCmd.Flags().StringP("output", "o", "", "File di output per salvare il changelog")
	changelogCmd.Flags().StringP("format", "f", "markdown", "Formato del changelog: markdown, teams")
	changelogCmd.Flags().BoolP("include-subtasks", "s", false, "Includi i sub-task nel changelog")
}
