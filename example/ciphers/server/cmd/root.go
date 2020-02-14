package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/lloydmeta/tasques/example/ciphers/server/config"
	"github.com/lloydmeta/tasques/example/ciphers/server/routing"
)

var (
	configFile string

	appConfig config.App

	defaultConfigPaths = []string{
		".",
		"./config",
		"/app/config",
	}
	rootCmd = &cobra.Command{
		Use:   "server",
		Short: "Messages and Ciphers",
		Long:  `CipherQueuer that stashes messages and puts jobs in for ciphering them up.`,
		Run: func(cmd *cobra.Command, args []string) {
			server := routing.Server{Config: appConfig}
			server.Run()
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (by default, looks in [%v] for 'tasques.yaml')", defaultConfigPaths))
}

// Executes the root command, which is to run the server
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func initConfig() {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("server")
		for _, p := range defaultConfigPaths {
			viper.AddConfigPath(p)
		}
	}

	viper.SetEnvPrefix("example")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msgf("Using config file: %v", viper.ConfigFileUsed())
	} else {
		log.Fatal().Err(err).Msg("Failed to read the config file")
	}

	// Marshal it
	err := viper.Unmarshal(&appConfig)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
