# Broker

## Introduction

This repository contains the Broker implemented in Go language. This version is an alternative to the [Python version](https://github.com/cloud-ai-ufcg/simulator-experiments). The structure of this version provides an _entrypoint_ function that allows it to be called by a main controller or to be executed by an API.

## How to run the Broker

As mentioned in the last section, you can execute the Broker using two methods. Let's talk about them in the next sections.

### Running by CLI

This section describes two methods to run it. The Broker can be executed using its `main.go` script or through a function that can be called by another Go module.<br>
For the first method, you need to define two paths in `main.go`: a CSV file containing the data to be submitted and a YAML file representing the configuration information to set up the Broker execution. You can use two example files that are available in the `example/` directory, which will always be up-to-date with the Broker version. To run in this case, use the following command: `go run cmd/main.go`, and the Broker will use the paths you define to execute correctly.<br>
For the second method, you must have a function in another module that will call the _entrypoint_ present in the Broker. The `Run` function, provided in the `broker.go` script, receives two arguments: the first is `event_data`, representing CSV data with the events; the second is `config_yaml`, containing the configuration file in YAML format. But pay attention! This function receives these arguments as pointers of the `os.File` type.

### Running by API

You can also run the Broker using an API version. In this case, the submission process may be executed through a route that will be described in the next sections. You must run the following command to start the HTTP server: `go run cmd/main.go --api`. <br>
At this point, the HTTP server will be listening on port `8080`, and you can make requests to its routes.

### Using a container

In this repository, there are two Dockerfiles available. The first is a file that allows you to execute the Broker in CLI mode (its name is just `Dockerfile`), and the second is a file that allows you to execute the Broker in API mode (`Dockerfile.api`). <br>
The Broker running in CLI mode inside a container will need to have the input data before you build the image. You can create a `data/` directory and set the path to your CSV data stored there. The Broker in API mode does not need the input data in your container, because you need to send this data using the route. To build the image, you can run:

```bash
docker build -f <Dockerfile-name> -t broker:latest .
```

In both cases, you need to mount the kubeconfig file into the container. This file is important to allow the Broker to make submissions to your cluster. This file is located in a directory created by Kubernetes on your file system. You can find it at `~/.kube/<config-file>`. The container must be able to connect to the cluster; for this, you need to set a network that allows the Broker to send requests to the cluster API server. <br>
Finally, run the following command to execute the Broker image that you built:

```bash
docker run -p 8080:80 --network <target-network> -v $HOME/.kube/karmada.config:/root/.kube/karmada.config broker:latest
```

**Note**: The network is the context that allows the connection with your cluster. If you are using a local cluster, you can set this parameter as `host`. The `-p` flag is optional and will be necessary when you run the Broker API image.

## Routes

* The `/broker/` route contains a *POST* HTTP method for starting the Broker submission process. The body of the request is exemplified in an example file located in the `example/` directory.