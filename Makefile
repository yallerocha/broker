.PHONY: all build run run-api clean

all: build run-api 

# builds the project and uses the 'bin/broker' directory as an output 
build:
	go build -o bin/broker cmd/main.go

# run the Broker in CLI mode
run:
	@echo "-- CLI MODE --"

	./bin/broker

# run the Broker in API mode
run-api:
	@echo "-- API MODE --"
	./bin/broker --api

# cleans the files created by the execution process
clean:
	go clean
	rm -rf ./bin/broker

# Show this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'