package main

import (
	"log"
	"os"

	"github.com/cloud-ai-ufcg/broker/broker"
)

func main() {
	f, err := os.Open("/home/joselima/Documentos/broker/examples/project_config_example.yaml")

	if err != nil {
		log.Fatalln(err)
	}

	defer f.Close()

	csv, err := os.Open("/home/joselima/Documentos/broker/examples/event_data_example.csv")

	if err != nil {
		log.Fatalln(err)
	}

	defer csv.Close()

	broker.Run(csv, f)
}
