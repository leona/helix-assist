# helix-assist

![Build Status](https://github.com/leona/helix-assist/actions/workflows/release.yml/badge.svg)
![GitHub Release](https://img.shields.io/github/v/release/leona/helix-assist)

A Go port of the [helix-gpt](https://github.com/leona/helix-gpt) LSP server, providing LLM code completions and actions tailored specifically for the Helix editor's LSP spec. This port serves as a more efficient, lightweight alternative using significantly less memory and resolving timeout issues and inconsistencies. Support is limited to OpenAI and Anthropic, with zero external dependencies.

## Features

- **Code Completions**: AI-powered code suggestions as you type
- **Code Actions**: Built-in commands for code improvement
  - Resolve diagnostics
  - Improve code
  - Refactor from comment

## Supported Providers

- **OpenAI** (default)
- **Anthropic**

## Installation

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/leona/helix-assist/releases), or install directly:

```bash
# Linux AMD64 example
wget https://github.com/leona/helix-assist/releases/latest/download/helix-assist-linux-amd64
chmod +x helix-assist-linux-amd64
sudo mv helix-assist-linux-amd64 /usr/local/bin/helix-assist
```

Binaries are available for Linux and macOS (both AMD64 and ARM64).

### Building from Source

```bash
# Install directly from GitHub
go install github.com/leona/helix-assist/cmd/helix-assist@latest

# Or clone and build manually
git clone https://github.com/leona/helix-assist.git
cd helix-assist

# Build for current platform
make build

# Install to $GOPATH/bin
make install

# Build for specific platform
make linux-amd64
make darwin-arm64

# Build for all platforms
make build-all
```

The binary will be created in the `build/` directory (or `$GOPATH/bin` with `make install` or `go install`).

## Helix Configuration

Add to `~/.config/helix/languages.toml`:

```toml
[language-server.helix-assist]
command = "helix-assist"
# Optional
args = ["--handler", "anthropic", "--num-suggestions", "2"]

[[language]]
name = "go"
language-servers = ["gopls", "helix-assist"]

[[language]]
name = "typescript"
language-servers = ["typescript-language-server", "helix-assist"]

[[language]]
name = "python"
language-servers = ["pylsp", "helix-assist"]
```
## Usage

1. Start Helix and open a file
2. Type a trigger character (`{`, `(`, or space) to get completions
3. Manually trigger the completion list with `Ctrl + X` to see suggestions
4. Select code and press `Space + a` to see code actions

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HANDLER` | `openai` | Provider: `openai` or `anthropic` |
| `OPENAI_API_KEY` | - | OpenAI API key |
| `OPENAI_MODEL` | `gpt-4.1-mini` | OpenAI model for completions |
| `OPENAI_ENDPOINT` | `https://api.openai.com/v1` | OpenAI API endpoint |
| `ANTHROPIC_API_KEY` | - | Anthropic API key |
| `ANTHROPIC_MODEL` | `claude-sonnet-4-5` | Anthropic model |
| `ANTHROPIC_ENDPOINT` | `https://api.anthropic.com` | Anthropic API endpoint |
| `DEBOUNCE` | `200` | Debounce delay in milliseconds |
| `TRIGGER_CHARACTERS` | `{`\|\|`(`\|\|` ` | Completion triggers (separated by `\|\|`) |
| `NUM_SUGGESTIONS` | `1` | Number of completion suggestions |
| `LOG_FILE` | `~/.cache/helix-assist.log` | Log file path |
| `FETCH_TIMEOUT` | `15000` | API request timeout (ms) |
| `ACTION_TIMEOUT` | `15000` | Code action timeout (ms) |
| `COMPLETION_TIMEOUT` | `15000` | Completion timeout (ms) |

## Debugging

Monitor helix-assist activity by tailing the log file:

```bash
tail -f ~/.cache/helix-assist.log
```

