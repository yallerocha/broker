package broker

import (
	"os"

	eventhandler "github.com/cloud-ai-ufcg/broker/internal/event-handler"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"gopkg.in/yaml.v3"
)

// Entrypoint to run the broker.
// it receives a CSV file and a yaml file representing the config file.
func Run(event_data *os.File, config_yaml *os.File) {
	config, df := process_parameters(event_data, config_yaml)

	utils.Configure_logger()
	initialize_context(config)

	eventhandler.Handler(df)
}

// Entrypoint to run the broker by the api execution.
// it receives a data frame and a config struct.
func Run_from_api(event_data *dataframe.DataFrame, config utils.Config) {
	utils.Configure_logger()
	initialize_context(config)

	eventhandler.Handler(*event_data)
}

// Retrieves the kubeconfig and Namespace defined by the config file.
// It loads this information as variables to be used in multiple contexts.
func initialize_context(config utils.Config) {
	utils.Set_context_path(config.KubeConfig)
	utils.Set_context_namespace(config.Namespace)
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
	utils.Log_fatal("Failed to decode the yaml.", err)

	return out
}

// Uses the dataframe module to process the event_data to a dataframe
func process_csv(event_data *os.File) dataframe.DataFrame {
	df := dataframe.ReadCSV(event_data)
	return df
}
