# KzLogViewer

**KzLogViewer** (by Karoza) is an ultra-lightweight, fully local, Kibana-inspired log viewer for
Docker, delivered as a single terminal application written in Go. It aggregates, displays, and lets
you search the logs of every running container on your machine in real time, without shipping them
anywhere, without Elasticsearch, without Logstash, and without any background agent.

Repository: [github.com/karozadev/KzLogViewer](https://github.com/karozadev/KzLogViewer)

[![ci](https://github.com/karozadev/KzLogViewer/actions/workflows/ci.yml/badge.svg)](https://github.com/karozadev/KzLogViewer/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/karozadev/KzLogViewer?include_prereleases)](https://github.com/karozadev/KzLogViewer/releases)
[![license](https://img.shields.io/github/license/karozadev/KzLogViewer)](LICENSE)
[![go report card](https://goreportcard.com/badge/github.com/karozadev/KzLogViewer)](https://goreportcard.com/report/github.com/karozadev/KzLogViewer)

---

## Why KzLogViewer

`docker logs` does not scale past one container. Full observability stacks (Elasticsearch, Logstash,
Kibana) solve this but are heavy to run just to tail logs on a developer machine. KzLogViewer sits in
between: it streams directly from the Docker Engine API into a single low-footprint binary, and gives
you a Kibana-style search and density view entirely in your terminal.

- **Native Docker connection.** Talks directly to `/var/run/docker.sock` through the official Docker
  Go SDK. No agent, no sidecar, no shipping data off the machine.
- **Kibana-inspired search bar.** Filter the live stream with plain text, an AND-of-keywords query,
  or a full regular expression.
- **Expandable, field-by-field detail view.** Toggle between a compact synthetic list and a detailed
  view that breaks a parsed JSON log line down field by field.
- **Density heatmap.** A one-line ANSI histogram at the top of the screen shows log volume and error
  spikes per minute, the same way a Kibana dashboard would.
- **Smart parser.** Automatic JSON detection, multiline stack trace reassembly, and severity-based
  coloring (`DEBUG`, `INFO`, `WARN`, `ERROR`).
- **Self-updating.** KzLogViewer checks GitHub Releases on startup and can update itself in place
  with `kzlogviewer update`, printing the cumulative changelog of every release skipped over.

## Installation

### One-liner (Linux / macOS)

```sh
curl -fsSL https://raw.githubusercontent.com/karozadev/KzLogViewer/main/install.sh | bash
```

or, with `wget`:

```sh
wget -qO- https://raw.githubusercontent.com/karozadev/KzLogViewer/main/install.sh | bash
```

The script detects your OS and architecture, downloads the matching release archive from
[GitHub Releases](https://github.com/karozadev/KzLogViewer/releases), and installs the `kzlogviewer`
binary to `/usr/local/bin` (or `$HOME/.local/bin` if that is not writable).

### Manual download

Download the archive for your platform from the
[latest release](https://github.com/karozadev/KzLogViewer/releases/latest), extract it, and place the
`kzlogviewer` binary on your `PATH`. Windows binaries are published as `.zip` archives.

### From source

```sh
go install github.com/karozadev/KzLogViewer/cmd/kzlogviewer@latest
```

## Usage

```sh
kzlogviewer            # launch the terminal UI
kzlogviewer update      # show the changelog and install the latest release
kzlogviewer version     # print version information
kzlogviewer help        # show usage
```

A Docker daemon reachable at `/var/run/docker.sock` (or `$DOCKER_HOST`) is required. No configuration
file is needed to get started.

`kzlogviewer update` looks up every release newer than the one you are running, not just the latest
one. If you skip several versions at once, it prints their changelogs concatenated, oldest first,
before downloading and installing the newest matching release.

### Interface overview

```
┌ KzLogViewer - 4 containers - 812/2450 entries ─────────────────────────────┐
│ ......:::---===++***##%%@@%%##**++===---:::......:::---===+++****##%%@@   │  <- density heatmap
├──────────────────────────────────────────────────────────────────────────┤
│ [text | all containers] search> timeout                                    │  <- search bar
├──────────────────────────────────────────────────────────────────────────┤
│ 14:32:01 INFO  api              request completed in 42ms                  │
│ 14:32:01 WARN  worker           queue depth above threshold (120)          │
│ 14:32:02 ERROR api              connection timeout after 30s               │  <- selected row
│ 14:32:02 INFO  db               checkpoint complete                        │
│ 14:32:03 DEBUG worker           picked up job 8f2a1c                       │
├──────────────────────────────────────────────────────────────────────────┤
│ streaming | / search  tab mode  c container  enter detail  p pause  q quit │  <- status bar
└──────────────────────────────────────────────────────────────────────────┘
```

Pressing `enter` on a selected row switches to the detailed, field-by-field view, similar to
expanding a document in Kibana's Discover panel:

```
┌ detail ──────────────────────────────────────────────────────────────────┐
│ timestamp     2024-01-02T14:32:02.104Z                                    │
│ container     api                                                         │
│ stream        stderr                                                      │
│ level         ERROR                                                       │
│ format        json                                                        │
│                                                                            │
│ fields                                                                    │
│   request_id  8f2a1c-91                                                   │
│   route       /v1/orders                                                  │
│   status      504                                                         │
│                                                                            │
│ message                                                                   │
│ connection timeout after 30s                                              │
└────────────────────────────────────────────────────────────────────────────┘
```

### Keybindings

| Key           | Action                                             |
|---------------|-----------------------------------------------------|
| `/`           | Focus the search bar                                 |
| `tab`         | Cycle the search mode (text -> keyword -> regexp)     |
| `enter`       | Run the search (while typing) / toggle detail view    |
| `esc`         | Cancel search input / back to the list view           |
| `up` / `k`    | Move selection up                                     |
| `down` / `j`  | Move selection down                                   |
| `g` / `Home`  | Jump to the oldest visible entry                       |
| `G` / `End`   | Jump to the newest entry and resume auto-scroll        |
| `c`           | Cycle the container filter                            |
| `p`           | Pause / resume the live stream                        |
| `q` / `ctrl+c`| Quit                                                   |

### The search engine

The search bar mirrors the ergonomics of Kibana's query bar with three interchangeable modes,
cycled with `tab`:

- **text** - a case-insensitive substring match against the raw log line.
- **keyword** - every whitespace-separated word in the query must appear in the line (an implicit
  AND), useful for narrowing down without worrying about exact phrasing.
- **regexp** - the query is compiled as a Go regular expression and matched against the raw line,
  for precise structured queries (e.g. `status=5\d\d`).

The container filter (`c`) and the search query compose: you can narrow to one container and search
within it at the same time.

## How it works

KzLogViewer is built around a strict Hexagonal (Ports and Adapters) architecture so the domain logic
never depends on Docker, Bubbletea, or the network:

```
cmd/kzlogviewer            composition root: wires adapters to services and starts the TUI
internal/core/domain       LogEntry, Container, HeatmapBucket - pure business types
internal/core/ports        LogSource, LogParser, QueryEngine, UpdateChecker, Applier interfaces
internal/core/services      Aggregator, SearchEngine, HeatmapBuilder - the application use cases
internal/adapters/docker    ports.LogSource implemented on the official Docker Go SDK
internal/adapters/parser    ports.LogParser: JSON detection, severity, multiline grouping
internal/adapters/tui       Bubbletea/Lipgloss terminal interface
internal/adapters/updater   ports.UpdateChecker / ports.Applier against the GitHub Releases API
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full architecture rationale, the branching model, and
the commit conventions used in this repository.

## Building and testing

```sh
go build ./...
go vet ./...
go test ./internal/... -race -cover
```

The test suite keeps coverage of `internal/...` (domain, ports, services, and adapters) at or above
80%, enforced in CI.

## License

KzLogViewer is released under the [MIT License](LICENSE).
