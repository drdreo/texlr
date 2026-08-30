package texlr

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

const usageText = `Texlr builds polished, self-contained LaTeX handoff documents.

Usage:
  texlr build <input.tex> --pdf <output.pdf> --source <bundle-dir> [options]
  texlr validate <input.tex> [options]
  texlr version

Build options:
  --pdf PATH       PDF output path (required)
  --source PATH    self-contained source bundle directory (required)
  --log PATH       copy the complete build log to PATH
  --force          replace existing output paths
  --json           emit a machine-readable result

Validate options:
  --log PATH       copy the validation log to PATH
  --force          replace an existing log path
  --json           emit a machine-readable result
`

// Run executes the Texlr CLI and returns a process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, _ = io.WriteString(stdout, usageText)
		return 0
	}

	switch args[0] {
	case "build":
		return runBuild(ctx, args[1:], stdout, stderr, version)
	case "validate":
		return runValidate(ctx, args[1:], stdout, stderr, version)
	case "version", "--version":
		_, _ = fmt.Fprintf(stdout, "texlr %s\n", version)
		return 0
	default:
		asJSON := jsonRequested(args[1:])
		if asJSON {
			return writeCLIError(result{Command: args[0]}, "unknown_command", fmt.Sprintf("unknown command %q", args[0]), true, stdout, stderr)
		}
		_, _ = fmt.Fprintf(stderr, "texlr: unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	asJSON := jsonRequested(args)
	if len(args) == 0 || isHelp(args[0]) {
		if asJSON {
			return writeCLIError(result{Command: "build"}, "usage_error", "build requires an input document", true, stdout, stderr)
		}
		_, _ = io.WriteString(stderr, usageText)
		return 2
	}

	options := buildOptions{inputPath: args[0], version: version}
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.pdfPath, "pdf", "", "PDF output path")
	flags.StringVar(&options.sourcePath, "source", "", "source bundle directory")
	flags.StringVar(&options.logPath, "log", "", "build log output path")
	flags.BoolVar(&options.force, "force", false, "replace existing outputs")
	flags.BoolVar(&options.asJSON, "json", false, "emit JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return writeCLIError(requestedResult("build", options), "usage_error", err.Error(), asJSON, stdout, stderr)
	}
	if flags.NArg() != 0 || options.pdfPath == "" || options.sourcePath == "" {
		return writeCLIError(requestedResult("build", options), "usage_error", "build requires one input, --pdf, and --source", asJSON, stdout, stderr)
	}

	value, err := build(ctx, options)
	return finish(value, err, options.asJSON, stdout, stderr)
}

func runValidate(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	asJSON := jsonRequested(args)
	if len(args) == 0 || isHelp(args[0]) {
		if asJSON {
			return writeCLIError(result{Command: "validate"}, "usage_error", "validate requires an input document", true, stdout, stderr)
		}
		_, _ = io.WriteString(stderr, usageText)
		return 2
	}

	options := buildOptions{inputPath: args[0], validateOnly: true, version: version}
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.logPath, "log", "", "validation log output path")
	flags.BoolVar(&options.force, "force", false, "replace an existing log")
	flags.BoolVar(&options.asJSON, "json", false, "emit JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return writeCLIError(requestedResult("validate", options), "usage_error", err.Error(), asJSON, stdout, stderr)
	}
	if flags.NArg() != 0 {
		return writeCLIError(requestedResult("validate", options), "usage_error", "validate accepts exactly one input", asJSON, stdout, stderr)
	}

	value, err := build(ctx, options)
	return finish(value, err, options.asJSON, stdout, stderr)
}

func finish(value result, err error, asJSON bool, stdout, stderr io.Writer) int {
	writer := stdout
	if err != nil && !asJSON {
		writer = stderr
	}
	if writeErr := writeResult(writer, value, asJSON); writeErr != nil {
		_, _ = fmt.Fprintf(stderr, "texlr: could not write result: %v\n", writeErr)
		return 1
	}
	if err != nil {
		return 1
	}
	return 0
}

func requestedResult(command string, options buildOptions) result {
	return result{
		Command:    command,
		PDFPath:    options.pdfPath,
		SourcePath: options.sourcePath,
		LogPath:    options.logPath,
	}
}

func writeCLIError(value result, kind, message string, asJSON bool, stdout, stderr io.Writer) int {
	value.Error = &errorResult{Kind: kind, Message: message}
	writer := stderr
	if asJSON {
		writer = stdout
	}
	if err := writeResult(writer, value, asJSON); err != nil {
		_, _ = fmt.Fprintf(stderr, "texlr: could not write result: %v\n", err)
		return 1
	}
	return 2
}

func jsonRequested(args []string) bool {
	for _, argument := range args {
		if argument == "--json" {
			return true
		}
	}
	return false
}

func isHelp(value string) bool {
	return value == "help" || value == "--help" || value == "-h"
}

func absolutePath(value string) (string, error) {
	path, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", value, err)
	}
	return filepath.Clean(path), nil
}
