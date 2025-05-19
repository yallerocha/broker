package main

import (
	"log"
	"os"

	"github.com/cloud-ai-ufcg/broker/broker"
)

func main() {
	csv_path := "/home/joselima/Documentos/broker/examples/project_config_example.yaml"
	yaml_path := "/home/joselima/Documentos/broker/data/unified_events2.csv"

	f, err := os.Open(csv_path)

	if err != nil {
		log.Fatalln(err)
	}

	defer f.Close()

	csv, err := os.Open(yaml_path)

	if err != nil {
		log.Fatalln(err)
	}

	defer csv.Close()

	broker.Run(csv, f)
}
