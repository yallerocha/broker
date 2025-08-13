package main

import (
	"log"
	"os"

	"github.com/cloud-ai-ufcg/broker/broker"
	"github.com/cloud-ai-ufcg/broker/internal/api/router"
	"github.com/cloud-ai-ufcg/broker/pkg/utils"
)

// main is the entrypoint of the Broker application.
// It initializes utilities and determines whether to run in API or CLI mode.
func main() {
	utils.Init()

	if utils.Get_action_selected() {
		api_init()
	} else {
		cli_init()
	}
}

// api_init sets up and starts the HTTP API router for the broker.
func api_init() {
	router.Init_router()
}

// cli_init prepares and runs the broker in command-line interface mode.
// It opens the necessary event data CSV and configuration YAML files.
func cli_init() {
	csv_path := "examples/event_data_example.csv"
	yaml_path := "examples/project_config_example.yaml"

	config_yaml, err := os.Open(yaml_path)
	if err != nil {
		log.Fatalln(err)
	}
	defer config_yaml.Close()


	csv, err := os.Open(csv_path)
	if err != nil {
		log.Fatalln(err)
	}
	defer csv.Close()

	broker.Run(csv, config_yaml)
}