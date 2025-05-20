# Broker
## Introduction
This repository contains the Broker implemented in Go language. This version is an alternative to the [Python version](https://github.com/cloud-ai-ufcg/simulator-experiments). The structure of this version provides an _entrypoint_ function that allows it to be called by a main controller.

## How to run the Broker
To run this Broker from the main script of another Go module, you must use the `Run` function provided in the `broker.go` script. This function receives two arguments: the first is `event_data`, representing CSV data with the events; the second is `config_yaml`, containing the configuration file in YAML format. In this repository, we provide two example files to test the arguments of this function, both located in the `example/` directory.

### About the configuration file
As mentioned in the last section, this repository contains an example configuration file in YAML format. It defines the important values required to run this Broker correctly. You must ensure that the `config_yaml` contains the items expected by the Broker, as shown in the example file.

### Isolated Test
If you do not have a main controller to run the Broker but still want to run for testing purposes, you can use the `main.go` script located in the `cmd/` directory. For this execution, you will need a CSV file with the events and a configuration file. The configuration file in the `example/` directory is a good choice for the testing process, as it will be kept up to date. After preparing these files, you must update the paths defined in the `main.go` script.

If you do not have a main controller to run the Broker but still want to run it for testing purposes, you can use the main.go script located in the cmd/ directory. For this execution, you will need a CSV file with the events and a configuration file. The configuration file in the example/ directory is a good choice for the testing process, as it will be kept up to date. After preparing these files, you must update the file paths defined in the main.go script.