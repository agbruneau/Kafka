# Kafka Crash Course

This project demonstrates a simple Kafka setup using Docker Compose, with a Python producer and consumer.

## Prerequisites

- Docker and Docker Compose
- Python 3
- `pip` for installing Python dependencies

## Setup

1. **Start the Kafka broker:**

   ```bash
   docker-compose up -d
   ```

2. **Install the Python dependency:**

   ```bash
   pip3 install confluent-kafka
   ```

## Usage

1. **Run the consumer:**

   Open a terminal and run the following command to start the consumer. The consumer will wait for messages on the `orders` topic.

   ```bash
   python3 tracker.py
   ```

2. **Run the producer:**

   Open another terminal and run the following command to send a sample message to the `orders` topic.

   ```bash
   python3 producer.py
   ```

   You should see the message appear in the consumer's terminal.

## Kafka Commands

Here are some useful commands for interacting with Kafka:

- **List all topics:**

  ```bash
  docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092
  ```

- **Describe a topic:**

  ```bash
  docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic orders
  ```

- **View all events in a topic:**

  ```bash
  docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
  ```
