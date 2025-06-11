# Trasher

A high-performance file generation tool that creates files of specified sizes with configurable data patterns using concurrent workers for optimal performance.

## Features

- **High Performance**: Concurrent worker pool architecture for maximum throughput
- **Multiple Data Patterns**: Support for random, sequential, zero, and mixed data patterns
- **Flexible Size Parsing**: Human-readable size formats (KB, MB, GB, TB, PB)
- **Progress Reporting**: Real-time progress with throughput and ETA calculations
- **Data Integrity**: Automatic SHA-256 checksum generation and verification
- **Graceful Shutdown**: Signal handling with cleanup on interruption
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Installation

### Using Make (Recommended)

```bash
git clone https://github.com/maxkimambo/trasher.git
cd trasher
make build
```

The binary will be created at `bin/trasher`.

### Using Go directly

```bash
go build -o trasher main.go
```

## Usage

### Basic Usage

```bash
./bin/trasher --size <SIZE> --output <FILE> [OPTIONS]
```

### Required Flags

- `--size, -s`: Size of file to generate (e.g., "1GB", "500MB", "2TB")
- `--output, -o`: Output file path

### Optional Flags

- `--pattern, -p`: Data pattern (default: "random")
  - `random`: Cryptographically secure random data
  - `sequential`: Sequential byte patterns (0-255 repeating)
  - `zero`: All zero bytes
  - `mixed`: Combination of different patterns
- `--workers, -w`: Number of worker goroutines (default: CPU cores)
- `--chunk-size, -c`: Size of data chunks per worker (default: "64MB")
- `--force, -f`: Overwrite existing files without confirmation
- `--verbose, -v`: Enable verbose output with detailed progress
- `--help, -h`: Show help message
- `--version`: Show version information

## Examples

### Generate a 1GB random file

```bash
./bin/trasher --size 1GB --output large_file.dat
```

**Output:**
```
Successfully generated large_file.dat
```

### Generate with verbose progress reporting

```bash
./bin/trasher --size 500MB --output test.dat --pattern random --verbose
```

**Output:**
```
Generating file: test.dat
Size: 500MB (524288000 bytes)
Pattern: random
Workers: 12
Chunk size: 64MB (67108864 bytes)

[==============================] | 100.00% | 1.2 GB/s | ETA: 0s | Elapsed: 0s | Written: 500.00 MB / 500.00 MB
                                                                                
Completed 500.00 MB in 0s (average 1.34 GB/s)

File generation completed successfully!
Output file: test.dat
Checksum file: test.dat.checksum.txt
```

### Generate a sequential pattern file

```bash
./bin/trasher --size 100MB --output sequential.dat --pattern sequential --verbose
```

**Output:**
```
Generating file: sequential.dat
Size: 100MB (104857600 bytes)
Pattern: sequential
Workers: 12
Chunk size: 64MB (67108864 bytes)

[==============================] | 100.00% | 2.1 GB/s | ETA: 0s | Elapsed: 0s | Written: 100.00 MB / 100.00 MB
                                                                                
Completed 100.00 MB in 0s (average 2.08 GB/s)

File generation completed successfully!
Output file: sequential.dat
Checksum file: sequential.dat.checksum.txt
```

### Generate with custom workers and chunk size

```bash
./bin/trasher --size 2GB --output custom.dat --workers 8 --chunk-size 128MB --verbose
```

**Output:**
```
Generating file: custom.dat
Size: 2GB (2147483648 bytes)
Pattern: random
Workers: 8
Chunk size: 128MB (134217728 bytes)

[==================>           ] | 65.50% | 890.2 MB/s | ETA: 1s | Elapsed: 1s | Written: 1.31 GB / 2.00 GB
```

### Generate zero-filled file

```bash
./bin/trasher --size 50MB --output zeros.dat --pattern zero
```

**Output:**
```
Successfully generated zeros.dat
```

### Force overwrite existing file

```bash
./bin/trasher --size 10MB --output existing.dat --force
```

**Output:**
```
Successfully generated existing.dat
```

## Size Formats

Trasher supports various human-readable size formats:

| Format | Description | Example |
|--------|-------------|---------|
| B | Bytes | `1024B` |
| KB | Kilobytes (1024 bytes) | `512KB` |
| MB | Megabytes (1024² bytes) | `100MB` |
| GB | Gigabytes (1024³ bytes) | `5GB` |
| TB | Terabytes (1024⁴ bytes) | `2TB` |
| PB | Petabytes (1024⁵ bytes) | `1PB` |

Decimal formats are also supported: `1000B`, `1.5GB`, `0.5TB`

## Data Patterns

### Random Pattern
- Cryptographically secure random data
- Best for testing with realistic data
- Uses Go's `crypto/rand` package

### Sequential Pattern
- Repeating sequence of bytes (0-255)
- Predictable pattern for testing
- Useful for compression testing

### Zero Pattern
- All bytes set to zero
- Fastest generation
- Useful for sparse file testing

### Mixed Pattern
- Combination of different patterns in chunks
- Provides varied data characteristics
- Good for comprehensive testing

## Output Files

Trasher generates two files:

1. **Data File**: The main file with generated content
2. **Checksum File**: SHA-256 checksum for integrity verification (`.checksum.txt`)

### Checksum File Format

```
# SHA-256 Checksum for test.dat
# Generated: 2024-06-11 15:30:45
# File Size: 104857600 bytes
# Pattern: random

a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456  test.dat
```

## Performance

Trasher is optimized for high performance:

- **Concurrent Workers**: Parallel data generation using worker pools
- **Buffer Pooling**: Efficient memory reuse to reduce GC pressure
- **Streaming Writes**: Direct file writing without intermediate buffering
- **Progress Reporting**: Non-blocking progress updates

### Typical Performance

| File Size | Pattern | Throughput (avg) |
|-----------|---------|------------------|
| 100MB | Random | 1-2 GB/s |
| 100MB | Sequential | 2-3 GB/s |
| 100MB | Zero | 3-5 GB/s |
| 1GB | Random | 800MB-1.5GB/s |
| 1GB | Zero | 2-4 GB/s |

*Performance varies based on hardware, storage type, and system load.*

## Development

### Building

```bash
# Build for current platform
make build

# Build with debug info
make dev

# Build for all platforms
make release
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Cleaning

```bash
# Clean build artifacts and test files
make clean
```

### Available Make Targets

```bash
make help
```

**Output:**
```
Available targets:
  build         - Build the application (default)
  test          - Run tests
  test-coverage - Run tests with coverage report
  clean         - Clean build artifacts and test files
  deps          - Download and tidy dependencies
  fmt           - Format code
  lint          - Lint code (requires golangci-lint)
  run           - Build and run the application
  dev           - Build with debug info
  release       - Build for multiple platforms
  help          - Show this help message
```

## Signal Handling

Trasher supports graceful shutdown on interruption:

```bash
# Start generation
./bin/trasher --size 10GB --output large.dat --verbose

# Press Ctrl+C during generation
^C
```

**Output:**
```
Generating file: large.dat
Size: 10GB (10737418240 bytes)
Pattern: random
Workers: 12
Chunk size: 64MB (67108864 bytes)

[===========>               ] | 35.2% | 1.1 GB/s | ETA: 6s | Elapsed: 3s | Written: 3.78 GB / 10.00 GB
Received signal terminated, shutting down gracefully...
Operation interrupted at 35.20% completion
Written: 3.78 GB / 10.00 GB
Partial file saved to: large.dat
Cleaning up resources...
                                                                                
Completed 3.78 GB in 3s (average 1.26 GB/s)

Error: operation cancelled
```

## Requirements

- Go 1.19 or later
- Sufficient disk space for target file size
- Write permissions to output directory

## License

MIT License - see LICENSE file for details.

