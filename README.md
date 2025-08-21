# HolyDB

This is my version of "holyDB", a database system that might include a different architecture.

## Getting Started

### Prerequisites

- Go 1.24.5 or later

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/garder500/holydb.git
   cd holydb
   ```

2. Build the application:
   ```bash
   make build
   # or
   go build .
   ```

### Usage

Run HolyDB:
```bash
./holydb
```

Show help:
```bash
./holydb help
```

Show version:
```bash
./holydb version
```

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Other Commands

- `make fmt` - Format code
- `make vet` - Vet code
- `make clean` - Clean build artifacts
- `make help` - Show all available make targets

## Project Structure

```
.
├── cmd/holydb/          # Command-line interface
├── pkg/db/              # Public database packages
├── internal/config/     # Internal configuration
├── main.go              # Application entry point
├── go.mod               # Go module definition
├── Makefile             # Build automation
└── README.md            # This file
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
