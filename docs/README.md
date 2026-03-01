# Kairo Documentation

Documentation for Kairo - Go CLI wrapper for Claude/Qwen Code with X25519 encryption and audit logging.

## Quick Links

| Resource                                              | Description                  |
| ----------------------------------------------------- | ---------------------------- |
| [User Guide](guides/user-guide.md)                    | Installation and basic usage |
| [Development Guide](guides/development-guide.md)      | Build, test, contribute      |
| [Architecture](architecture/README.md)                | System design and diagrams   |
| [Troubleshooting](troubleshooting/README.md)          | Common issues and solutions  |
| [Contributing](contributing/README.md)                | Contribution workflow        |
| [Configuration Reference](reference/configuration.md) | Config files and options     |
| [Provider Reference](reference/providers.md)          | Built-in providers           |

## Guides

### For Users

- [User Guide](guides/user-guide.md) - Installation, setup, and command reference

### For Developers

- [Development Guide](guides/development-guide.md) - Setup, testing, and contribution workflow

## Architecture

- [System Architecture](architecture/README.md) - Overview, components, data flow
- [Wrapper Scripts](architecture/wrapper-scripts.md) - Security design for token passing
- [ADRs](architecture/adr/) - Architecture decision records

## Reference

- [Configuration Reference](reference/configuration.md) - File formats and options
- [Provider Reference](reference/providers.md) - Built-in providers

## Operations

- [Troubleshooting](troubleshooting/README.md) - Common issues and solutions

## Contributing

- [Contributing Guide](contributing/README.md) - How to contribute
- [Changelog](../CHANGELOG.md) - Version history

## Documentation Structure

```text
docs/
├── README.md                 # This file
├── architecture/             # System design
│   ├── README.md             # Architecture overview
│   └── wrapper-scripts.md    # Security design
├── guides/                   # User and developer guides
│   ├── user-guide.md         # Basic usage
│   └── development-guide.md  # Development setup
├── reference/                # Reference documentation
│   ├── configuration.md      # Config files
│   └── providers.md          # Provider details
├── troubleshooting/          # Problem solving
│   └── README.md
└── contributing/             # Contribution
    └── README.md
```
