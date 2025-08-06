# README.md

# Cardano Tx Sync

This project provides a reliable and scalable service to sync blocks from a Cardano node via Ogmios, parse transactions, and publish them to relevant Kafka topics based on configurable mappings.

## Features

- **Real-time Sync**: Connects to Ogmios's chainsync mini-protocol to receive blocks as they are added to the chain.
- **Dynamic Topic Mapping**: Publishes transaction data to Kafka topics based on mappings for output addresses or asset policy IDs stored in a database.
- **Flexible Message Encoding**: Supports multiple message formats (e.g., full JSON, simple ID) on a per-mapping basis.
- **Reliable Checkpointing**: Keeps track of the last synced block to ensure seamless resumption after a restart.
- **Rollback Handling**: Correctly handles blockchain rollbacks by invalidating data and notifying downstream services.
- **Management API**: Provides HTTP endpoints to manage topic mappings and control the sync starting point.
- **Scalable**: Designed to be scalable by leveraging Kafka for message distribution.
- **Containerized**: Comes with a `Dockerfile` and `docker-compose.yml` for easy setup and deployment.

## Project Structure

```
.
├── cmd/                # Main application
├── config/             # Configuration loading
├── internal/
│   ├── api/            # HTTP API server
│   ├── chainsync/      # Ogmios chainsync logic
│   ├── encoder/        # Message encoders (JSON, Simple, etc.)
│   ├── handler/        # Block processing and Kafka publishing
│   ├── kafka/          # Kafka producer wrapper
│   ├── model/          # Application data models
│   └── storage/        # Database interaction (PostgreSQL)
├── go.mod
├── go.sum
├── Dockerfile
└── docker-compose.yml
```

## Getting Started

### Prerequisites

- [Docker](https://www.docker.com/get-started) and [Docker Compose](https://docs.docker.com/compose/install/)
- A running Cardano node with Ogmios enabled and accessible.

### 1. Configuration

Copy the example configuration file and edit it to match your environment.

```bash
cp config.yaml.example config.yaml
```

Update `config.yaml` with your Ogmios endpoint, Kafka broker addresses, and database connection details. The default `docker-compose.yml` sets up Kafka and Postgres, so the default settings should work for a local setup. You will need to provide your Ogmios endpoint.

### 2. Build and Run with Docker Compose

The easiest way to run the entire stack (the bridge application, Kafka, and PostgreSQL) is with Docker Compose.

```bash
docker-compose up --build
```

This command will:
1. Build the Go application Docker image.
2. Start containers for Zookeeper, Kafka, and PostgreSQL.
3. Start the `ogmios-kafka-bridge` application container.

The application will start syncing from the Cardano network and the API will be available on `http://localhost:8080`.

### 3. Using the API

You can interact with the service using its REST API.

#### Add a new mapping

**Endpoint**: `POST /mappings`

**Body Examples**:
```json
{
    "type": "address",
    "key": "addr1q8...your_address",
    "topic": "my-awesome-dapp-transactions",
    "encoder": "DEFAULT"
}
```json
{
    "type": "policy_id",
    "key": "your_policy_id",
    "topic": "my-nft-project-mints",
    "encoder": "SIMPLE"
}
```
The `encoder` field is optional and defaults to `DEFAULT`. Supported values are `DEFAULT`, `SIMPLE`, and `DANOGO`.

#### Remove a mapping

**Endpoint**: `DELETE /mappings/:id`

Replace `:id` with the numerical ID of the mapping you want to remove.

#### Set a custom sync start point

**Endpoint**: `POST /sync/start`

This will clear all existing checkpoints and restart the sync from the specified point. Use with caution.

**Body**:
```json
{
    "slot": 65000000,
    "hash": "ab...cdef"
}
```

## Development

### Running Tests

To run the unit tests for the project:

```bash
go test ./...
