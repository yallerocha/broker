package main

import (
	"log"
	"os"

	"github.com/cloud-ai-ufcg/broker/broker"
)

func main() {
	csv_path := "your-path"
	yaml_path := "your-path"

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
