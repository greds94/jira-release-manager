package cmd

import (
	"fmt"
	"os"

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
}

// Execute esegue il comando root
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Legge le variabili d'ambiente
	viper.SetEnvPrefix("") // Nessun prefisso, così legge JIRA_URL direttamente
	viper.AutomaticEnv()

	// Opzionalmente, puoi anche leggere da un file .env
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Ignora errore se il file non esiste
	_ = viper.ReadInConfig()
}
