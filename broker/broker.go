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
	log_setting()
	config, df := process_parameters(event_data, config_yaml)

	eventhandler.Handler(config, df, DefaultLogger)
}

func process_parameters(event_data *os.File, config_yaml *os.File) (utils.Config, dataframe.DataFrame) {
	config := process_yaml(config_yaml)
	df := process_csv(event_data)

	return config, df

}

func process_yaml(config_yaml *os.File) utils.Config {
	var out utils.Config
	decoder := yaml.NewDecoder(config_yaml)

	err := decoder.Decode(&out)
	utils.Exit_if_err(err, "Failed to decode the yaml.")

	return out
}

func process_csv(event_data *os.File) dataframe.DataFrame {
	df := dataframe.ReadCSV(event_data)
	return df
}

func log_setting() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

	DefaultLogger = slog.New(handler)
	DefaultLogger.With("prefix", "[BROKER]")
	slog.SetDefault(DefaultLogger)
}
