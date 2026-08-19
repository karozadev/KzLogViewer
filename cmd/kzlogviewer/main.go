// Command kzlogviewer is the composition root of KzLogViewer: it wires the
// Docker, parser, updater and TUI adapters to the core application
// services and starts the terminal UI (or runs the "update" subcommand).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/karozadev/KzLogViewer/internal/adapters/docker"
	"github.com/karozadev/KzLogViewer/internal/adapters/parser"
	"github.com/karozadev/KzLogViewer/internal/adapters/tui"
	"github.com/karozadev/KzLogViewer/internal/adapters/updater"
	"github.com/karozadev/KzLogViewer/internal/config"
	"github.com/karozadev/KzLogViewer/internal/core/ports"
	"github.com/karozadev/KzLogViewer/internal/core/services"
	"github.com/karozadev/KzLogViewer/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "update":
			os.Exit(runUpdate())
		case "version", "-v", "--version":
			fmt.Printf("kzlogviewer %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
			return
		case "-h", "--help", "help":
			printUsage()
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}
	os.Exit(runTUI())
}

func printUsage() {
	fmt.Println(`KzLogViewer - a lightweight, local, Kibana-inspired log viewer for Docker.

Usage:
  kzlogviewer            Launch the terminal UI
  kzlogviewer update      Show the changelog and install the latest release
  kzlogviewer version     Print version information
  kzlogviewer help        Show this help message`)
}

func runTUI() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	source, err := docker.NewFromEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to Docker: %v\n", err)
		return 1
	}

	aggregator := services.NewAggregator(source, parser.New(), 0)
	entries, errs := aggregator.Run(ctx)

	cfg := config.FromEnvironment()
	var checker ports.UpdateChecker
	if !cfg.DisableUpdateCheck {
		checker = updater.NewGitHubChecker()
	}

	model := tui.NewModel(entries, errs, services.NewSearchEngine(), services.NewHeatmapBuilder(0), checker, version.Version)

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "kzlogviewer: %v\n", err)
		return 1
	}
	return 0
}

func runUpdate() int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	checker := updater.NewGitHubChecker()
	fmt.Println("Checking for new KzLogViewer releases...")
	releases, err := checker.ReleasesSince(ctx, version.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check for updates: %v\n", err)
		return 1
	}

	if len(releases) == 0 {
		fmt.Printf("Already up to date (%s).\n", version.Version)
		return 0
	}

	printChangelog(releases)

	target := releases[len(releases)-1]
	if target.DownloadURL == "" {
		fmt.Fprintf(os.Stderr, "no release asset for your platform in %s; download it manually from https://github.com/karozadev/KzLogViewer/releases/tag/%s\n", target.Version, target.Version)
		return 1
	}

	fmt.Printf("Updating from %s to %s...\n", version.Version, target.Version)
	if err := updater.NewBinaryApplier().Apply(ctx, target); err != nil {
		fmt.Fprintf(os.Stderr, "apply update: %v\n", err)
		return 1
	}
	fmt.Printf("Updated to %s.\n", target.Version)
	return 0
}

// printChangelog prints the changelog of every release being skipped over,
// oldest first, so jumping several versions at once shows the cumulative
// history rather than only the latest release's notes.
func printChangelog(releases []ports.Release) {
	fmt.Println("\nChangelog:")
	for _, r := range releases {
		fmt.Printf("\n## %s\n", r.Version)
		notes := strings.TrimSpace(r.Notes)
		if notes == "" {
			notes = "(no release notes)"
		}
		fmt.Println(notes)
	}
	fmt.Println()
}
