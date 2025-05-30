package broker

import (
	"log/slog"
	"os"

	eventhandler "github.com/cloud-ai-ufcg/broker/internal/event-handler"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"gopkg.in/yaml.v3"
)

var DefaultLogger *slog.Logger

// Entrypoint to run the broker
// it receives a CSV file and a yaml file representing the config file
func Run(event_data *os.File, config_yaml *os.File) {
	config, df := process_parameters(event_data, config_yaml)

	log_setting()
	initialize_context(config)

	eventhandler.Handler(config, df, DefaultLogger)
}

func initialize_context(config utils.Config) {
	utils.Set_context_path(config.KubeConfig)
}

// Process the parameters passed by the main script.
func process_parameters(event_data *os.File, config_yaml *os.File) (utils.Config, dataframe.DataFrame) {
	config := process_yaml(config_yaml)
	df := process_csv(event_data)

	return config, df

}

// Uses the 'struct' defined in the models.go
// to convert the yaml to Config
func process_yaml(config_yaml *os.File) utils.Config {
	var out utils.Config
	decoder := yaml.NewDecoder(config_yaml)

	err := decoder.Decode(&out)
	utils.Exit_if_err(err, "Failed to decode the yaml.")

	return out
}

// Uses the dataframe module to process the event_data to a dataframe
func process_csv(event_data *os.File) dataframe.DataFrame {
	df := dataframe.ReadCSV(event_data)
	return df
}

// Set the logger
func log_setting() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

	DefaultLogger = slog.New(handler)
	DefaultLogger.With("prefix", "[BROKER]")
	slog.SetDefault(DefaultLogger)
}
