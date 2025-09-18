package broker

import (
	"os"

	eventhandler "github.com/cloud-ai-ufcg/broker/internal/handlers"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"gopkg.in/yaml.v3"
)

// Run is the entrypoint to start the broker via CLI.
// It processes event data from a CSV file and configuration from a YAML file.
func Run(event_data *os.File, config_yaml *os.File) {
	config, df := process_parameters(event_data, config_yaml)

	utils.Configure_logger()
	initialize_context(config)

	// Handles events in CLI mode.
	eventhandler.Handler(df, eventhandler.ModeCLI, config)
}

// Run_from_api is the entrypoint to start the broker via API execution.
// It receives a dataframe of event data, a configuration struct, and an execution mode.
func Run_from_api(event_data *dataframe.DataFrame, config utils.Config, mode string) {
	utils.Configure_logger()
	initialize_context(config)

	// Handles events based on the provided API execution mode (e.g., "init", "simulation").
	eventhandler.Handler(*event_data, mode, config)
}

// initialize_context retrieves the KubeConfig path and Namespace from the provided configuration.
// It sets these as global variables to be used across different contexts.
func initialize_context(config utils.Config) {
	utils.Set_context(config)
}

// process_parameters processes input files (CSV for event data, YAML for configuration).
// It converts them into a Config struct and a Gota DataFrame.
func process_parameters(event_data *os.File, config_yaml *os.File) (utils.Config, dataframe.DataFrame) {
	config := process_yaml(config_yaml)
	df := process_csv(event_data)

	return config, df
}

// process_yaml converts a YAML configuration file into a utils.Config struct.
// It uses the 'yaml.v3' decoder for parsing.
func process_yaml(config_yaml *os.File) utils.Config {
	var out utils.Config
	decoder := yaml.NewDecoder(config_yaml)

	err := decoder.Decode(&out)
	utils.Log_fatal("Failed to decode the YAML.", err) // Improved error message

	return out
}

// process_csv uses the 'go-gota/dataframe' module to process a CSV event data file into a DataFrame.
func process_csv(event_data *os.File) dataframe.DataFrame {
	df := dataframe.ReadCSV(event_data)
	return df
}
