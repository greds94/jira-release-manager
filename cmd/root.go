package cmd

import (
	"fmt"
	"os"
	"strings"

	"jira-release-manager/internal/jira"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "jira-release-manager",
	Short: "Un tool CLI per gestire release e changelog di progetti Jira",
	Long: `Jira Release Manager è un tool da linea di comando che permette di:
- Visualizzare i ticket della prossima release pianificata
- Generare changelog formattati per le comunicazioni
- Avere una overview completa di ticket e sub-task`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		projectKey, err = cmd.Flags().GetString("project")
		if err != nil {
			return err
		}
		if projectKey == "" {
			if cmd.Name() == "help" || strings.HasPrefix(cmd.Name(), "__") {
				return nil
			}
			return fmt.Errorf("il flag --project (-p) è obbligatorio")
		}

		jiraClient, err = jira.NewClient()
		if err != nil {
			return fmt.Errorf("errore nella creazione del client Jira: %w", err)
		}

		return nil
	},
}

var (
	projectKey string
	jiraClient *jira.Client
)

// Execute esegue il comando root
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("project", "p", "", "Chiave del progetto Jira (es. PROJ)")
}

func initConfig() {
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	_ = viper.ReadInConfig()
}
