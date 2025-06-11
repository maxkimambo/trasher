# Trasher CLI Tool - Product Requirements Document

## Executive Summary

Trasher is a command-line tool written in Go designed to generate test data files of specified sizes in binary format. The tool supports multiple data generation patterns and leverages parallel processing for optimal performance, capable of generating files up to 10PB in size.

## Product Overview

### Purpose
Trasher addresses the need for generating large volumes of test data for performance testing, storage benchmarking, and system validation scenarios where realistic file sizes and data patterns are required.

### Target Users
- Software engineers conducting performance testing
- DevOps engineers benchmarking storage systems
- QA engineers requiring large test datasets
- System administrators testing backup and recovery systems

## Functional Requirements

### Core Features

#### 1. File Size Specification
- **Requirement**: Accept file size via command-line argument `--size`
- **Supported Units**: Bytes (B), Kilobytes (KB), Megabytes (MB), Gigabytes (GB), Terabytes (TB), Petabytes (PB)
- **Range**: 1 byte to 10 petabytes
- **Format Examples**: 
  - `100MB`
  - `1.5GB` 
  - `10TB`
  - `2PB`

#### 2. Data Pattern Generation
- **Requirement**: Support multiple data generation patterns via `--pattern` argument
- **Supported Patterns**:
  - `random`: Generate pseudo-random data using Go's crypto/rand
  - `sequential`: Generate sequential byte patterns (0x00, 0x01, 0x02, ...)
  - `mixed`: Randomly alternate between random data chunks and zero-filled chunks
  - `zero`: Generate zero-filled data (equivalent to /dev/zero)

#### 3. Output File Management
- **Requirement**: Accept output file path via `--output` argument
- **Functionality**:
  - Create output file if it doesn't exist
  - Overwrite existing files (with optional confirmation)
  - Validate write permissions to target directory
  - Support absolute and relative paths

#### 4. Parallel Processing
- **Requirement**: Utilize multiple goroutines for data generation
- **Implementation**:
  - Automatically detect optimal number of workers based on CPU cores
  - Allow manual worker count specification via `--workers` flag
  - Implement work distribution across goroutines
  - Ensure thread-safe file writing operations

### Command-Line Interface

#### Basic Usage
```bash
trasher --size <SIZE> --pattern <PATTERN> --output <OUTPUT_PATH>
```

#### Full Command Specification
```bash
trasher [OPTIONS]

Options:
  --size, -s        Size of file to generate (required)
  --pattern, -p     Data pattern to generate (required)
                    Options: random, sequential, mixed, zero
  --output, -o      Output file path (required)
  --workers, -w     Number of worker goroutines (optional, default: CPU cores)
  --chunk-size, -c  Size of data chunks per worker (optional, default: 64MB)
  --force, -f       Overwrite existing files without confirmation
  --verbose, -v     Enable verbose output
  --help, -h        Show help information
  --version         Show version information
```

#### Example Commands
```bash
# Generate 100MB random data file
trasher --size 100MB --pattern random --output /tmp/testfile.bin

# Generate 10GB mixed pattern file with 8 workers
trasher --size 10GB --pattern mixed --output /tmp/large.bin --workers 8

# Generate 1TB sequential data with verbose output
trasher --size 1TB --pattern sequential --output /storage/test.bin --verbose
```

## Technical Requirements

### Performance Requirements
- **Memory Efficiency**: Maximum memory usage should not exceed 1GB regardless of output file size
- **Throughput**: Target minimum 1GB/s write speed on modern SSDs
- **Scalability**: Linear performance improvement with additional CPU cores
- **Progress Reporting**: Real-time progress updates for files > 1GB

### Error Handling
- **Invalid Size Format**: Graceful handling with clear error messages
- **Insufficient Disk Space**: Pre-flight disk space validation
- **Permission Errors**: Clear error messages for write permission issues
- **Interrupted Operations**: Graceful shutdown with cleanup on SIGINT/SIGTERM
- **File System Limitations**: Handle file size limits of target file system

### Platform Support
- **Primary**: Linux (amd64, arm64)
- **Secondary**: macOS (amd64, arm64)
- **Tertiary**: Windows (amd64)

### Dependencies
- **Runtime**: Go 1.21 or higher
- **External Dependencies**: Minimize external dependencies, prefer standard library
- **Build Tools**: Standard Go toolchain
- **CLI Libraries**: Use `cobra` for command-line argument parsing

## Implementation Specifications

### Architecture Design

#### Core Components
1. **CLI Parser**: Command-line argument parsing and validation
2. **Size Parser**: Human-readable size string parsing
3. **Pattern Generators**: Pluggable data pattern generation
4. **Worker Pool**: Goroutine management for parallel processing
5. **File Writer**: Thread-safe file writing coordination
6. **Progress Reporter**: Real-time progress tracking and display
7. **Verification Mode**: Data integrity verification capabilities

#### Data Flow
1. Parse and validate command-line arguments
2. Validate output path and disk space
3. Initialize worker pool based on CPU cores or user specification
4. Distribute work chunks across workers
5. Generate data according to specified pattern, each chunk should represent this pattern e.g random, mixed, zero
6. Write data to output file in coordinated manner
7. Report progress and completion status
8. Generate a verification checksum for the generated file, checksum should be calculated as data is being generated, chunk checksums can also be used for faster verification, output checksum into separate file checksum.txt
9. Handle graceful shutdown on cancellation signals

### Memory Management
- **Chunk-based Processing**: Process data in configurable chunks (default 64MB)
- **Buffer Reuse**: Implement buffer pools to minimize garbage collection
- **Streaming Write**: Write data as it's generated to avoid memory accumulation

### Concurrency Model
- **Producer-Consumer**: Workers generate data, single writer coordinates file output
- **Channel Communication**: Use Go channels for worker coordination
- **Graceful Shutdown**: Implement context-based cancellation for clean shutdowns

## Quality Assurance

### Testing Strategy
- **Unit Tests**: Comprehensive coverage for all core functions
- **Integration Tests**: End-to-end testing with various file sizes and patterns
- **Performance Tests**: Benchmarking against target performance metrics
- **Error Scenario Tests**: Validation of error handling paths

### Acceptance Criteria
- Successfully generate files from 1MB to 10GB in various patterns
- Handle all specified data patterns correctly
- Achieve target performance metrics
- Graceful error handling for all failure scenarios
- Cross-platform compatibility verification (low priority for Windows)

## Future Enhancements

### Phase 2 Features
- **Resume Capability**: Resume interrupted large file generation


## Success Metrics

### Primary Metrics
- **Generation Speed**: Average throughput in GB/s
- **Memory Efficiency**: Peak memory usage during operation
- **Error Rate**: Percentage of successful completions
- **User Adoption**: Number of downloads and active users

### Secondary Metrics
- **Cross-platform Compatibility**: Success rate across supported platforms
- **Large File Success**: Completion rate for files > 1TB
- **Performance Scaling**: Throughput improvement with additional cores

## Risk Assessment

### Technical Risks
- **File System Limitations**: Some file systems may not support very large files
- **Memory Pressure**: Potential memory issues on resource-constrained systems
- **Platform Differences**: Varying performance characteristics across platforms

### Mitigation Strategies
- Implement pre-flight file system capability checks
- Provide configurable memory usage limits
- Extensive cross-platform testing and optimization