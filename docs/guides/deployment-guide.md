# Deployment Guide

Deployment instructions for Kairo CLI.

## Installation Methods

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh
```

### Build from Source

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
make build
sudo mv dist/kairo /usr/local/bin/
```

### Homebrew (macOS/Linux)

```bash
brew install dkmnx/tap/kairo
```

## System Requirements

| Requirement    | Minimum               | Recommended   |
| -------------- | --------------------- | ------------- |
| Go version     | 1.25                  | 1.25+         |
| OS             | Linux, macOS, Windows | Latest        |
| Architecture   | amd64, arm64          | amd64         |
| Disk space     | 10 MB                 | 50 MB         |

## Path Configuration

Add to PATH:

**Linux/macOS (bash):**

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

**Linux/macOS (fish):**

```bash
fish_add_path -g $HOME/.local/bin
```

**Windows:**

```powershell
# Add to User PATH via System Properties > Environment Variables
%USERPROFILE%\.local\bin
```

Verify installation:

```bash
kairo version
```

## Configuration

### Default Location

| OS        | Path                                                |
| --------- | --------------------------------------------------- |
| Linux     | `$XDG_CONFIG_HOME/kairo` or `~/.config/kairo`       |
| macOS     | `~/Library/Application Support/kairo`               |
| Windows   | `%APPDATA%\kairo`                                   |

### File Permissions

All sensitive files use 0600 permissions:

| File          | Purpose                              |
|---------------|--------------------------------------|
| `config`      | Provider configurations (YAML)       |
| `secrets.age` | Encrypted API keys                   |
| `age.key`     | Encryption private key               |

### Environment Variables

| Variable           | Purpose                         | Default             |
|--------------------|---------------------------------|---------------------|
| `KAIRO_CONFIG_DIR` | Override config directory       | Platform default    |
| `KAIRO_UPDATE_URL` | Custom update check URL         | GitHub Releases API |

## Shell Completion

Enable shell completion for better UX:

**Bash:**

```bash
kairo completion bash >> ~/.bashrc
source ~/.bashrc
```

**Zsh:**

```bash
kairo completion zsh > ~/.zsh/completion/_kairo
# Add to ~/.zshrc:
fpath+=~/.zsh/completion
autoload -U compinit
compinit
```

**Fish:**

```bash
kairo completion fish > ~/.config/fish/completions/kairo.fish
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Install Kairo
  run: |
    curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh
    echo "$HOME/.local/bin" >> $GITHUB_PATH
```

### Docker

```dockerfile
FROM golang:1.25-alpine as builder
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk add --no-cache curl
COPY --from=builder /app/dist/kairo /usr/local/bin/
RUN kairo version
```

## Verification

After installation, verify setup:

```bash
# Check version
kairo version

# List providers (after setup)
kairo list

# Test provider connectivity
kairo status
```

## Updates

### Manual Update

```bash
kairo update
```

### Auto-Update Notification

`kairo version` checks for updates and notifies if new version available.

### Disable Update Check

```bash
# Set to any value to disable
export KAIRO_SKIP_UPDATE_CHECK=1
```

## Troubleshooting

### "command not found: kairo"

Binary not in PATH. Add to PATH as shown above.

### Permission Denied

```bash
chmod +x ~/.local/bin/kairo
```

### Configuration Not Found

Run `kairo setup` to initialize configuration.

See [Troubleshooting Guide](../troubleshooting/README.md) for more solutions.
