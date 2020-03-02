package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"

	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/infra/server"
)

var (
	configFile string
	appConfig  config.App
	logFile    *os.File

	defaultConfigPaths = []string{
		".",
		"./config",
		"/app/config",
	}
	rootCmd = &cobra.Command{
		Use:   "tasques",
		Short: "tasques is a Task manager.",
		Long:  `tasques is a Task manager backed by Elasticsearch`,
		Run: func(cmd *cobra.Command, args []string) {
			components, err := server.NewComponents(&appConfig)
			if err != nil {
				log.Fatal().Err(err).Send()
			} else {
				components.Run()
			}
		},
	}
)

// Executes the root command, which is to run the server
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Send()
		closeLogFile()
	}
	defer closeLogFile()
}

func init() {
	cobra.OnInitialize(initConfig, configureLogging, configureApm)
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (by default, looks in [%v] for 'tasques.yaml')", defaultConfigPaths))
}

// initConfig reads the application config and sets it globally
func initConfig() {
	viper.AllowEmptyEnv(true)
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("tasques")
		for _, p := range defaultConfigPaths {
			viper.AddConfigPath(p)
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msgf("Using config file: %v", viper.ConfigFileUsed())
	} else {
		log.Fatal().Err(err).Msg("Failed to read the config file")
	}

	// Unmarshal it, UnmarshalKey doesn't play well with Env vars, hence
	// the top level wrapping in order to do namespacing in the config file
	var t config.TopLevel
	err := viper.Unmarshal(&t)

	if err != nil {
		log.Error().Err(err).Send()
		closeLogFile()
		os.Exit(1)
	}
	appConfig = t.Tasques.Server
}

// configureLogging configures the logger based on loaded config
// It assumes that the config has already been set and is non-nil
func configureLogging() {
	jsonLogging := false
	var file *string
	var level *zerolog.Level
	if appConfig.Logging != nil {
		if appConfig.Logging.Json != nil {
			jsonLogging = *appConfig.Logging.Json
		}
		if appConfig.Logging.File != nil {
			file = appConfig.Logging.File
		}
		if appConfig.Logging.Level != nil {
			parsedLevel, err := zerolog.ParseLevel(*appConfig.Logging.Level)
			if err != nil {
				log.Warn().
					Str("configured_level", *appConfig.Logging.Level).
					Str("will_use_level", zerolog.InfoLevel.String()).
					Msg("Invalid level configured, ignoring")
			} else {
				level = &parsedLevel
			}
		}
	}
	writeTo := os.Stderr // default
	if file != nil {
		f, err := os.OpenFile(*file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to open log file for writing.")
		}
		logFile = f
		writeTo = f
	}
	if !jsonLogging {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: writeTo})
	} else {
		log.Logger = log.Output(writeTo)
	}
	if level == nil {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(*level)
	}

}

func closeLogFile() {
	if logFile != nil {
		_ = logFile.Close()
	}
}

// configureApm configures APM by simply setting env vars if we override their
// values in our config
func configureApm() {
	if v := os.Getenv("ELASTIC_APM_SERVICE_NAME"); len(v) == 0 {
		if err := os.Setenv("ELASTIC_APM_SERVICE_NAME", "tasques"); err != nil {
			log.Fatal().Err(err).Send()
		}
	}
	if appConfig.ApmClient != nil {
		apmConf := *appConfig.ApmClient
		log.Info().Interface("apm_conf", apmConf).Msg("Configuring APM based on config file values")

		if apmConf.Address != nil {
			if err := os.Setenv("ELASTIC_APM_SERVER_URL", *apmConf.Address); err != nil {
				log.Fatal().Err(err).Send()
			}
		}
		if apmConf.SecretToken != nil {
			if err := os.Setenv("ELASTIC_APM_SECRET_TOKEN", *apmConf.SecretToken); err != nil {
				log.Fatal().Err(err).Send()
			}
		}

	}
	// re-init the global tracer
	tracerOptions := apm.TracerOptions{}
	apmTransport, err := transport.NewHTTPTransport()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tracerOptions.Transport = apmTransport
	if tracer, err := apm.NewTracerOptions(tracerOptions); err != nil {
		log.Fatal().Err(err).Send()
	} else {
		apm.DefaultTracer.Close()
		apm.DefaultTracer = tracer
	}
}
