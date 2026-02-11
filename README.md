# Broker

The Broker is the workload submission component responsible for creating, updating, and deleting Kubernetes resources (deployments and jobs) across multiple clusters. It processes event data from CSV files or API requests and applies them to the target clusters through Karmada or direct Kubernetes API calls.

## Overview

The Broker acts as the interface between the simulator's event-driven workload generation and the actual Kubernetes infrastructure. It translates simulation events into concrete Kubernetes resources, managing their lifecycle across clusters. The Broker ensures that workload specifications from the simulator are accurately deployed to the appropriate clusters at the right time.

### Event Types

The Broker processes three types of workload events:

**Create**: Deploys new workloads (deployments or jobs) to specified clusters

**Update**: Modifies existing workload specifications (resource requests, replicas, etc.)

**Delete**: Removes workloads from clusters

## Prerequisites

- Go 1.19 or higher
- Access to Kubernetes clusters (configured via kubeconfig)
- Karmada installed and configured (for multi-cluster management)
- Docker and Docker Compose (for containerized deployment)

## How to Run

> **Note:** For detailed setup and execution instructions, including infrastructure setup and complete workflow, please refer to the [main simulator README](../README.md). The Broker is typically run as part of the complete simulator environment via Docker Compose. It operates in API mode, listening on port 8080 for workload submission requests from the simulator's main controller.

When running as part of the simulator, the Broker:
- Automatically connects to configured Kubernetes clusters via mounted kubeconfig
- Processes workload events as they are generated during simulation
- Applies resources through Karmada for multi-cluster coordination
