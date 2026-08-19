# Contributing to KzLogViewer

Thank you for your interest in improving KzLogViewer. This document describes the branching model,
the commit convention, and the quality bar every change is expected to meet before it is merged.

## Project architecture

KzLogViewer follows a strict Hexagonal (Ports and Adapters) architecture:

- `internal/core/domain` holds the business entities (`LogEntry`, `Container`, `HeatmapBucket`, ...).
  It has zero dependency on Docker, Bubbletea, or any other technical detail.
- `internal/core/ports` defines the interfaces through which the core talks to the outside world
  (`LogSource`, `LogParser`, `QueryEngine`, `UpdateChecker`, `Applier`).
- `internal/core/services` implements the application use cases (log aggregation, search, heatmap
  aggregation) purely in terms of ports, so they can be unit tested with fakes and no real Docker
  daemon.
- `internal/adapters/*` implements the ports: `docker` (Docker Engine API), `parser` (JSON/severity
  detection), `tui` (Bubbletea/Lipgloss interface), `updater` (GitHub Releases API and binary
  replacement).

When contributing a new feature, ask which side of the boundary it belongs to. Business rules and
orchestration belong in `core`; anything that talks to a real external system belongs in an
adapter behind a port.

## Branching model: GitFlow

This repository uses a lightweight GitFlow:

- `main` always reflects the latest released (or releasable) state. Every tag (`vX.Y.Z`,
  `vX.Y.Z-beta.N`) is cut from `main` and triggers the release pipeline.
- `develop` is the integration branch for the next release. Feature branches target `develop`.
- `feature/<short-description>` branches are created from `develop` for new functionality
  (e.g. `feature/regexp-search`).
- `fix/<short-description>` branches are created from `develop` (or from `main` for a hotfix) for
  bug fixes.
- `release/<version>` branches are cut from `develop` when preparing a release, allowing final
  stabilization without blocking new feature work; they are merged into both `main` and `develop`.
- `hotfix/<short-description>` branches are created from `main` for urgent production fixes and are
  merged back into both `main` and `develop`.

Open a pull request from your branch into `develop` (or `main` for a hotfix). Rebase on top of the
target branch before requesting review to keep history linear.

## Conventional Commits

Every commit message must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Allowed types:

- `feat`: a new user-facing capability (e.g. `feat(search): support regexp query mode`)
- `fix`: a bug fix (e.g. `fix(docker): handle containers with no name`)
- `test`: adding or correcting tests, with no production code change
- `refactor`: a code change that neither fixes a bug nor adds a feature
- `docs`: documentation only changes
- `chore`: tooling, dependencies, CI configuration
- `perf`: a performance improvement

Keep the description in the imperative mood ("add", not "added" or "adds") and under 72 characters.
Reference the relevant issue in the footer when applicable (`Refs #42`).

Only commit work that is complete and tested. Do not commit half-finished features behind
placeholder code; open a draft pull request instead if you want early feedback.

## Quality bar before opening a pull request

1. **Format and vet**: `gofmt -l .` must report nothing; `go vet ./...` must be clean.
2. **Lint**: `golangci-lint run` must pass (the same configuration CI uses, see `.golangci.yml`).
3. **Tests**: `go test ./internal/... -race -cover` must pass. New code must be covered by unit
   tests; mock the relevant port (see `internal/core/services/aggregator_test.go` for an example of
   faking `ports.LogSource` and `ports.LogParser`) rather than requiring a live Docker daemon.
4. **Coverage**: the overall coverage of `internal/...` must stay at or above 80%, the threshold
   enforced by the `ci` workflow. `cmd/kzlogviewer` (the composition root that only wires adapters
   together) is intentionally excluded from the threshold, but should still be exercised manually.
5. **No emoji**: keep the codebase, TUI output, and documentation free of emoji and emoticons.

## Running the test suite locally

```
go build ./...
go vet ./...
gofmt -l .
go test ./internal/... -race -cover
```

## Running KzLogViewer locally

```
go run ./cmd/kzlogviewer
```

A running Docker daemon reachable at `/var/run/docker.sock` (or `$DOCKER_HOST`) is required to see
live containers.

## Reporting issues

Please include your OS, Docker version, the output of `kzlogviewer version`, and steps to reproduce
when filing a bug report.
