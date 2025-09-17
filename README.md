# Log Processing Pipeline

A high-performance log masking and normalization tool built in Go that processes large log files (tested with 500K+ lines) with minimal memory usage (~2MB) and sub-second performance.

## Overview

This project implements a sophisticated log processing pipeline designed for:
- **High-performance log masking**: Processes 500K+ lines in sub-second time
- **Memory efficiency**: Maintains constant ~2MB memory usage regardless of input size
- **Log contextualization**: Extracts patterns and context from log data using AI
- **Token labeling**: Identifies and labels different components within log entries

## Architecture

### Core Pipeline Components

**Reader** (`reader.go`): FileReader ingests log files line by line using buffered scanning, converting bytes to rune slices via UTF-8 decoding to avoid string allocations.

**Processor** (`maskConsumer.go`): MaskConsumer applies log masking by:
- Replacing alphanumeric characters with 'Y' tokens
- Masking nested content within brackets/quotes with 'X' tokens
- Compressing consecutive 'Y' tokens to reduce output size
- Supporting nested enclosing symbols: `[]`, `{}`, `<>`, `()`, `""`, `''`

**Writer** (`writer.go`): FileWriter outputs processed logs to files using buffered writing.

**Memory Management** (`runePool.go`): Custom RunePool provides bounded buffer pooling to eliminate garbage collection pressure and maintain constant memory usage.

### Advanced Components

**Contextualiser** (`contextualiser.go`): Uses AI (Braintrust) to analyze log patterns and extract contextual information from masked logs.

**Admin** (`admin.go`): Routes processed sentences between registered and unregistered channels for further processing.

**Labeller** (`labeller.go`): Labels tokens within sentences based on extracted context.

**Store** (`store.go`): Generic MemoryStore for key-value operations with reporting capabilities.

## Usage

### Basic Commands

```bash
# Run the program (processes logs from ./data/raw/mixed.log to ./data/results/data.log)
go run .

# Build the binary
go build

# Run tests
go test
```

### Performance Profiling

```bash
# Generate memory profile
go run . -memprofile mem.prof

# Generate CPU profile
go run . -cpuprofile cpu.prof

# Analyze profiles
go tool pprof mem.prof
go tool pprof cpu.prof

# Enable GC tracing
GODEBUG=gctrace=1 go run .
```

## Key Features

### Performance Optimizations

- **Rune-based processing**: Uses `[]rune` throughout the pipeline instead of strings to avoid allocations
- **Channel-based communication**: Goroutines communicate via buffered channels (typically 100 buffer size)
- **Buffer pooling**: Reuses rune buffers to prevent GC pressure during high-throughput processing
- **Streaming processing**: Processes logs incrementally rather than loading entire files into memory

### Data Flow

1. **FileReader** scans input file → rune buffers from pool
2. **MaskConsumer** processes buffers → applies masking logic
3. **Contextualiser** analyzes patterns → extracts context using AI
4. **Admin** routes sentences → separates registered/unregistered patterns
5. **Labeller** applies labels → identifies token types
6. **FileWriter** writes results → returns buffers to pool

All components run concurrently connected via channels.

## Performance Results

Through extensive optimization based on profiling:
- **Memory usage**: Reduced from 1.6GB to ~2MB through buffer reuse
- **Processing time**: Reduced from 11+ seconds to sub-second performance
- **Eliminated string allocations** in hot paths
- **Fixed race conditions** between concurrent workers and writers

The system achieves **562,473 lines per second** throughput while maintaining constant memory usage.

## Dependencies

- Go 1.25.0+
- [Braintrust Go SDK](https://github.com/braintrustdata/braintrust-go) for AI-powered contextualization
- Standard library packages for file I/O and concurrency

## Project Structure

```
├── main.go              # Entry point and pipeline coordination
├── reader.go            # File reading with buffered scanning
├── maskConsumer.go      # Log masking and token processing
├── writer.go            # Buffered file writing
├── runePool.go          # Memory pool for rune buffers
├── contextualiser.go    # AI-powered pattern analysis
├── admin.go            # Sentence routing and administration
├── labeller.go         # Token labeling based on context
├── store.go            # Generic key-value store
├── data/
│   ├── raw/            # Input log files
│   └── results/        # Processed output files
└── CLAUDE.md           # Development guidelines
```

## Development

This codebase prioritizes performance and memory efficiency. Key design decisions include:

- Zero-allocation processing through buffer reuse
- Concurrent pipeline architecture for maximum throughput
- Streaming approach to handle files larger than available memory
- AI integration for intelligent log pattern recognition

All optimizations are backed by profiling data to ensure measurable performance improvements.