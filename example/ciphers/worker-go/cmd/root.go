package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/lloydmeta/tasques/example/ciphers/worker-go/config"
	"github.com/lloydmeta/tasques/example/ciphers/worker-go/handlers"
	"github.com/lloydmeta/tasques/example/ciphers/worker-go/persistence"
	"github.com/lloydmeta/tasques/worker"
	config2 "github.com/lloydmeta/tasques/worker/config"
)

var (
	configFile     string
	workerId       config2.WorkerId
	randomFailures bool

	appConfig config.App

	defaultConfigPaths = []string{
		".",
		"./config",
		"/app/config",
	}
	rootCmd = &cobra.Command{
		Use:   "cipher-worker",
		Short: "Plain to Ciphered",
		Long:  `Worker that polls for plain messages and ciphers them`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Str("worker-id", string(appConfig.CipherWorker.TasquesWorker.ID)).Msg("Starting worker")
			esClient, err := appConfig.Elasticsearch.BuildClient()
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to build ES client")
			}

			messagesRepo := persistence.NewMessagesRepo(esClient)
			if randomFailures {
				log.Info().Bool("random_failures", randomFailures).Msg("Will randomly fail tasks")
			}
			cipherWorker := handlers.CipherWorker{
				Repo:           messagesRepo,
				RandomFailures: randomFailures,
			}

			workLoop := worker.NewWorkLoop(appConfig.CipherWorker.TasquesWorker, worker.QueuesToHandles{
				appConfig.CipherWorker.Queue: &cipherWorker,
			})

			if workLoop.Run() != nil {
				log.Fatal().Err(err).Msg("Error running Work loop")
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (by default, looks in [%v] for 'tasques.yaml')", defaultConfigPaths))
	rootCmd.PersistentFlags().BoolVarP(&randomFailures, "random-failures", "f", false, "Whether or not this worker should randomly fail")
	rootCmd.PersistentFlags().Var(&workerId, "worker-id", "What to use as the worker id")
	rootCmd.MarkPersistentFlagRequired("worker-id")
	cobra.OnInitialize(initConfig)
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
		viper.SetConfigName("worker")
		for _, p := range defaultConfigPaths {
			viper.AddConfigPath(p)
		}
	}

	// Bind the config value to the flag
	viper.BindPFlag("cipher_worker.tasques_worker.id", rootCmd.PersistentFlags().Lookup("worker-id"))

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
		log.Fatal().Err(err).Msg("Failed to unmarshal")
		os.Exit(1)
	}
}
