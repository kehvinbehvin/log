# Apache Kafka with Go - Comprehensive Guide

## Table of Contents
1. [Introduction](#introduction)
2. [Core Concepts](#core-concepts)
3. [Architecture](#architecture)
4. [Key Features and Use Cases](#key-features-and-use-cases)
5. [Ecosystem Components](#ecosystem-components)
6. [Performance and Scalability](#performance-and-scalability)
7. [Durability and Reliability](#durability-and-reliability)
8. [Setup and Configuration](#setup-and-configuration)
9. [Production Best Practices](#production-best-practices)
10. [Go Client Libraries](#go-client-libraries)
11. [Code Examples](#code-examples)
12. [Architectural Patterns](#architectural-patterns)
13. [Troubleshooting and Monitoring](#troubleshooting-and-monitoring)

## Introduction

Apache Kafka is an open-source distributed event streaming platform designed for building real-time data pipelines and streaming applications. Originally developed at LinkedIn and later open-sourced under the Apache Software Foundation, Kafka excels at processing large volumes of data in a scalable, fault-tolerant manner.

Kafka is ideal for use cases such as:
- Real-time analytics
- Data ingestion pipelines
- Event-driven architectures
- Log aggregation
- Microservices communication
- Stream processing

## Core Concepts

### Brokers
A **broker** is a server in the Kafka storage layer that stores event streams from producers and serves them to consumers. A Kafka cluster consists of multiple brokers, and each broker acts as a bootstrap server through which clients can discover the entire cluster.

### Topics
A **topic** is Kafka's fundamental unit of organization, acting as an append-only, immutable log of events. Producers write to topics and consumers read from topics, with events organized in time order and identified by offsets.

### Partitions
**Partitions** subdivide a topic into ordered, immutable sequences of records across which Kafka can parallelize reads and writes. Each partition is hosted on a single broker and replicated to others for fault tolerance.

### Producers
A **producer** is a client application that publishes records to Kafka topics. Producers can control partition assignment either via round-robin for load balancing or by key-based partitioning.

### Consumers
A **consumer** is a client application that subscribes to one or more topics and reads records in order by offset. Consumers track their offsets to support replaying or skipping records.

### Consumer Groups
A **consumer group** is a logical grouping of consumers cooperating to consume topic partitions in parallel. Kafka ensures each partition is consumed by only one group member at a time, enabling scalable processing with at-least-once delivery guarantees.

## Architecture

Kafka's brokers form a cluster where:
- The **control plane** manages metadata and partition assignments
- The **data plane** handles client produce and fetch requests
- Each partition's leader broker handles writes and replicates data to follower brokers

### ZooKeeper vs KRaft Mode

**Historical (pre-4.0):** Kafka relied on ZooKeeper for metadata management and leader election.

**KRaft Mode (4.0+):** Kafka introduces KRaft mode, embedding the controller within brokers and eliminating the ZooKeeper dependency.

### Internal Storage

- Each partition's log is segmented into files for efficient storage and retention
- Index files (offset and time indexes) enable fast record lookups
- Replication protocol ensures data durability through in-sync replicas (ISR)
- Leverages Linux page cache and `sendfile` system call for zero-copy data transfer

## Key Features and Use Cases

### Performance
- High throughput: Peak cluster throughput of ~605 MB/s
- Low latency: p99 latencies around 5 ms under 200 MB/s load
- Optimized through disk I/O, batching, and zero-copy networking

### Scalability
- Partition-based parallelism
- Horizontal scaling by adding brokers
- Linear performance scaling with cluster size

### Reliability
- Configurable replication and in-sync replicas
- Log retention policies
- Exactly-once semantics through idempotent producers and transactional APIs

### Integration
- Stream processing (Kafka Streams, ksqlDB)
- Connectors for external systems (Kafka Connect)

## Ecosystem Components

### Kafka Connect
A framework for building and running reusable connectors that move large data sets into and out of Kafka with support for transformations, distributed mode, and fault tolerance.

### Kafka Streams
A Java library for building stateful stream processing applications with high-level DSL for aggregations, joins, windowing, and exactly-once semantics.

### ksqlDB
A streaming SQL engine for real-time data processing directly on Kafka topics, supporting continuous queries, materialized views, and UDFs.

### Schema Registry
Centralized schema management and compatibility enforcement for Avro, JSON Schema, and Protobuf, enabling decoupled schema evolution.

### Control Center
A UI for cluster monitoring, topic and consumer group management, connectors, and alerting.

## Performance and Scalability

Kafka's design around sequential disk I/O and efficient networking yields:
- High throughput and low latency
- Zero-copy and page cache utilization
- Ability to saturate modern NVMe disks at hundreds of MB/s
- Partition-level parallelism for linear scaling
- Strong performance under multi-producer, multi-consumer workloads

## Durability and Reliability

### Data Durability
- Enforced by topics' replication factor and in-sync replica requirements
- Producers can set `acks=all` and brokers enforce `min.insync.replicas`
- Ensures messages are safely written before acknowledgment

### Exactly-Once Semantics
- Provided via idempotent producers and transactions
- Prevents duplicates and ensures atomic writes across multiple partitions
- KRaft mode improves metadata reliability

## Setup and Configuration

### Traditional Setup (pre-4.0)
```bash
# Download and extract Kafka
wget https://archive.apache.org/dist/kafka/3.7.0/kafka_2.13-3.7.0.tgz
tar -xzf kafka_2.13-3.7.0.tgz
cd kafka_2.13-3.7.0

# Start ZooKeeper
bin/zookeeper-server-start.sh config/zookeeper.properties

# Start Kafka broker
bin/kafka-server-start.sh config/server.properties

# Create a topic
bin/kafka-topics.sh --create --topic test --bootstrap-server localhost:9092 \
    --partitions 3 --replication-factor 1
```

### KRaft Mode Setup (4.0+)
Edit `server.properties`:
```properties
process.roles=broker,controller
controller.quorum.voters=0@localhost:9093
listeners=PLAINTEXT://:9092,CONTROLLER://:9093
```

Start Kafka:
```bash
bin/kafka-server-start.sh config/server.properties
```

### Important Configuration Parameters
- `broker.id`: Unique identifier for each broker
- `log.dirs`: Directory where log files are stored
- `num.partitions`: Default number of partitions for new topics
- `default.replication.factor`: Default replication factor for new topics
- `min.insync.replicas`: Minimum number of in-sync replicas
- `listeners`: Network addresses the broker binds to
- `advertised.listeners`: Addresses published to clients

## Production Best Practices

### Cluster Setup
- Deploy ≥3 brokers with replication factor ≥3
- Configure `min.insync.replicas`=2 for durability
- Distribute partitions evenly across brokers and racks

### Security
- Secure communication with TLS
- Authentication with SASL
- Enforce ACLs for authorization

### Performance Tuning
- **OS Level:**
  - Disable swap
  - Use performance CPU governor
  - Tune file descriptor limits
- **JVM Level:**
  - Use G1GC garbage collector
  - Allocate adequate heap memory
  - Monitor GC pause times

### Monitoring
- Monitor JMX metrics via Prometheus
- Watch under-replicated partitions
- Track ISR shrinkage
- Monitor request latencies
- Track consumer lag

### Data Management
- Configure retention and compaction policies
- Align with data lifecycle requirements
- Regular Kafka version upgrades

## Go Client Libraries

### 1. Sarama (IBM/Shopify)
Pure Go library with comprehensive Kafka support:
```go
import "github.com/IBM/sarama"
```
**Features:**
- Producers, consumers, admin APIs, consumer groups
- Compatibility guarantees across Kafka versions
- Pure Go implementation (no C dependencies)

### 2. Confluent Kafka Go
Go binding for librdkafka:
```go
import "github.com/confluentinc/confluent-kafka-go/kafka"
```
**Features:**
- High throughput and performance
- Transactional producers
- Balanced consumers
- Admin client functionality
- C library dependency (librdkafka)

### 3. kafka-go (Segment)
Go-native implementation:
```go
import "github.com/segmentio/kafka-go"
```
**Features:**
- Low-level primitives
- Readers/writers abstraction
- Flexible connection handling
- No C dependencies

## Code Examples

### Producer Example (Sarama)
```go
package main

import (
    "log"
    "github.com/IBM/sarama"
)

func main() {
    // Producer configuration
    config := sarama.NewConfig()
    config.Producer.Return.Successes = true
    config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas
    config.Producer.Retry.Max = 5
    
    // Create producer
    producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
    if err != nil {
        log.Fatal("Failed to create producer:", err)
    }
    defer producer.Close()
    
    // Send message
    message := &sarama.ProducerMessage{
        Topic: "test-topic",
        Key:   sarama.StringEncoder("user-123"),
        Value: sarama.StringEncoder("Hello from Go!"),
    }
    
    partition, offset, err := producer.SendMessage(message)
    if err != nil {
        log.Fatal("Failed to send message:", err)
    }
    
    log.Printf("Message stored in partition %d, offset %d", partition, offset)
}
```

### Consumer Example (Confluent Kafka Go)
```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
    // Consumer configuration
    consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers": "localhost:9092",
        "group.id":          "go-consumer-group",
        "auto.offset.reset": "earliest",
    })
    
    if err != nil {
        log.Fatal("Failed to create consumer:", err)
    }
    defer consumer.Close()
    
    // Subscribe to topics
    err = consumer.SubscribeTopics([]string{"test-topic"}, nil)
    if err != nil {
        log.Fatal("Failed to subscribe:", err)
    }
    
    // Handle shutdown gracefully
    sigchan := make(chan os.Signal, 1)
    signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
    
    run := true
    for run {
        select {
        case sig := <-sigchan:
            log.Printf("Caught signal %v: terminating", sig)
            run = false
            
        default:
            ev := consumer.Poll(100)
            if ev == nil {
                continue
            }
            
            switch e := ev.(type) {
            case *kafka.Message:
                log.Printf("Received message: %s = %s (partition=%d, offset=%d)",
                    string(e.Key), string(e.Value), e.TopicPartition.Partition, e.TopicPartition.Offset)
                
            case kafka.Error:
                log.Printf("Consumer error: %v", e)
                if e.Code() == kafka.ErrAllBrokersDown {
                    run = false
                }
            }
        }
    }
}
```

### Advanced Producer with Error Handling (kafka-go)
```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/segmentio/kafka-go"
)

func main() {
    // Configure writer
    w := &kafka.Writer{
        Addr:         kafka.TCP("localhost:9092"),
        Topic:        "test-topic",
        Balancer:     &kafka.LeastBytes{}, // Balance messages across partitions
        BatchTimeout: 10 * time.Millisecond,
        BatchSize:    100,
    }
    defer w.Close()
    
    // Send messages
    messages := []kafka.Message{
        {
            Key:   []byte("user-1"),
            Value: []byte("message 1"),
        },
        {
            Key:   []byte("user-2"), 
            Value: []byte("message 2"),
        },
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    err := w.WriteMessages(ctx, messages...)
    if err != nil {
        log.Fatal("Failed to write messages:", err)
    }
    
    log.Println("Messages sent successfully")
}
```

### Consumer Group Example with Manual Commit
```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/segmentio/kafka-go"
)

func main() {
    // Configure reader for consumer group
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:        []string{"localhost:9092"},
        Topic:          "test-topic",
        GroupID:        "go-consumer-group",
        StartOffset:    kafka.FirstOffset,
        CommitInterval: time.Second,
        MinBytes:       10e3, // 10KB
        MaxBytes:       10e6, // 10MB
    })
    defer r.Close()
    
    for {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        
        message, err := r.ReadMessage(ctx)
        cancel()
        
        if err != nil {
            log.Printf("Error reading message: %v", err)
            continue
        }
        
        log.Printf("Received: %s = %s (partition=%d, offset=%d)",
            string(message.Key), string(message.Value), 
            message.Partition, message.Offset)
        
        // Process message here
        processMessage(message)
        
        // Manual commit after processing
        if err := r.CommitMessages(context.Background(), message); err != nil {
            log.Printf("Failed to commit message: %v", err)
        }
    }
}

func processMessage(msg kafka.Message) {
    // Your message processing logic here
    log.Printf("Processing message: %s", string(msg.Value))
}
```

## Architectural Patterns

### 1. Event Sourcing
Persist domain events in Kafka and rebuild application state by replaying topics:
```go
// Event structure
type Event struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"`
    Data      json.RawMessage `json:"data"`
    Timestamp time.Time `json:"timestamp"`
}

// Store events in Kafka
func storeEvent(producer sarama.SyncProducer, event Event) error {
    eventData, _ := json.Marshal(event)
    message := &sarama.ProducerMessage{
        Topic: "events",
        Key:   sarama.StringEncoder(event.ID),
        Value: sarama.ByteEncoder(eventData),
    }
    _, _, err := producer.SendMessage(message)
    return err
}
```

### 2. CQRS (Command Query Responsibility Segregation)
Separate command (write) and query (read) models using Kafka for command events and materialized views for queries.

### 3. Log Aggregation
Centralize application logs and metrics into Kafka for processing and storage:
```go
type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Service   string    `json:"service"`
    Message   string    `json:"message"`
    Metadata  map[string]interface{} `json:"metadata"`
}
```

### 4. Microservices Event Bus
Decouple microservices communication through Kafka topics for commands and events.

### 5. Streaming ETL
Ingest, transform, and load data streams using Kafka Connect, Kafka Streams, or custom Go processors.

## Troubleshooting and Monitoring

### Key Metrics to Monitor

#### Broker Metrics
- Under-replicated partitions
- Request latencies (produce/fetch)
- Disk utilization
- GC pause durations
- Network I/O

#### Producer Metrics
- Request latency
- Request rate
- Error rate
- Batch size

#### Consumer Metrics
- Consumer lag per partition
- Commit rate
- Processing time
- Rebalance frequency

### Common Issues and Solutions

#### High Consumer Lag
```bash
# Check consumer lag
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
    --group my-group --describe
```

**Solutions:**
- Scale consumer instances
- Optimize message processing
- Tune consumer configuration
- Consider partitioning strategy

#### Rebalancing Issues
**Symptoms:** Frequent consumer group rebalances
**Solutions:**
- Tune `session.timeout.ms` and `heartbeat.interval.ms`
- Use incremental cooperative assignor
- Optimize consumer processing time

#### Disk Space Issues
**Symptoms:** Brokers running out of disk space
**Solutions:**
- Configure appropriate retention policies
- Monitor log segment sizes
- Implement log compaction for key-based topics

### Monitoring Setup Example

#### JMX Metrics Export
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'kafka'
    static_configs:
      - targets: ['localhost:9308']  # JMX exporter port
```

#### Key Alerts
```yaml
# alert.rules
groups:
  - name: kafka
    rules:
      - alert: KafkaUnderReplicatedPartitions
        expr: kafka_server_replicamanager_underreplicatedpartitions > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Kafka has under-replicated partitions"
          
      - alert: KafkaConsumerLag
        expr: kafka_consumer_lag_sum > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Kafka consumer lag is high"
```

### Debugging Tools

#### Kafka Console Tools
```bash
# List topics
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Describe topic
kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic my-topic

# Console consumer
kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic my-topic --from-beginning

# Consumer group details
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group my-group --describe
```

#### Log Analysis
- Monitor broker logs for errors and warnings
- Check client application logs for connection issues
- Analyze GC logs for performance bottlenecks

This comprehensive guide provides the foundation for understanding and implementing Apache Kafka solutions with Go. The combination of Kafka's robust streaming platform and Go's performance characteristics makes for powerful, scalable distributed systems.