package main

import (
	"log"
	"os"

	"github.com/cloud-ai-ufcg/broker/broker"
)

func main() {
	csv_path := "your-path"
	yaml_path := "your-path"

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
