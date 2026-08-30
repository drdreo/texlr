package texlr

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverDiagrams(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "main.tex"), `
\graphviz[width=\linewidth]{diagrams/system.dot}
\mermaid{diagrams/flow.mmd}
\graphviz{diagrams/system.dot}
`)

	items, err := discoverDiagrams(root, filepath.Join(root, "main.tex"))
	if err != nil {
		t.Fatalf("discover diagrams: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d diagrams, want 2", len(items))
	}
	if items[0].kind != "graphviz" || items[0].relSource != "diagrams/system.dot" {
		t.Fatalf("unexpected first diagram: %#v", items[0])
	}
	if items[1].kind != "mermaid" || items[1].relOutput != "diagrams/flow.mmd.pdf" {
		t.Fatalf("unexpected second diagram: %#v", items[1])
	}
}

func TestDiscoverDiagramsUsesOnlyActiveContent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "main.tex"), `
% \graphviz{diagrams/commented.dot}
\begin{lstlisting}
\mermaid{diagrams/example.mmd}
\end{lstlisting}
% \begin{lstlisting}
\graphviz{diagrams/still-active.dot}
% \end{lstlisting}
\input{sections/active}
`)
	writeTestFile(t, filepath.Join(root, "sections", "active.tex"), `\graphviz{diagrams/active.dot}`)
	writeTestFile(t, filepath.Join(root, "unused.tex"), `\mermaid{diagrams/unused.mmd}`)

	items, err := discoverDiagrams(root, filepath.Join(root, "main.tex"))
	if err != nil {
		t.Fatalf("discover diagrams: %v", err)
	}
	if len(items) != 2 || items[0].relSource != "diagrams/active.dot" || items[1].relSource != "diagrams/still-active.dot" {
		t.Fatalf("unexpected active diagrams: %#v", items)
	}
}

func TestBuildPublishesArtifacts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based fake tools are Unix-only")
	}
	toolDir := installFakeTools(t)
	t.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "handoff.tex"), `\documentclass{texlr}
\begin{document}
\graphviz{diagrams/system.dot}
\mermaid{diagrams/flow.mmd}
\end{document}
`)
	writeTestFile(t, filepath.Join(root, "diagrams", "system.dot"), "digraph G { a -> b }\n")
	writeTestFile(t, filepath.Join(root, "diagrams", "flow.mmd"), "flowchart LR\n  A --> B\n")

	artifacts := filepath.Join(root, "artifacts")
	options := buildOptions{
		inputPath:  filepath.Join(root, "handoff.tex"),
		pdfPath:    filepath.Join(artifacts, "handoff.pdf"),
		sourcePath: filepath.Join(artifacts, "source"),
		logPath:    filepath.Join(artifacts, "handoff.log"),
		version:    "test",
	}
	value, err := build(context.Background(), options)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !value.Success || len(value.Diagrams) != 2 {
		t.Fatalf("unexpected result: %#v", value)
	}
	for _, expected := range []struct {
		path string
		mode os.FileMode
	}{
		{options.pdfPath, 0o644},
		{options.logPath, 0o644},
		{options.sourcePath, 0o755},
	} {
		info, statErr := os.Stat(expected.path)
		if statErr != nil {
			t.Fatalf("stat %s: %v", expected.path, statErr)
		}
		if info.Mode().Perm() != expected.mode {
			t.Errorf("mode for %s = %o, want %o", expected.path, info.Mode().Perm(), expected.mode)
		}
	}
	for _, path := range []string{
		options.pdfPath,
		options.logPath,
		filepath.Join(options.sourcePath, "texlr.cls"),
		filepath.Join(options.sourcePath, "diagrams", "system.dot.pdf"),
		filepath.Join(options.sourcePath, "diagrams", "flow.mmd.pdf"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected artifact %s: %v", path, err)
		}
	}
	manifestContent, err := os.ReadFile(filepath.Join(options.sourcePath, "texlr-manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifestValue manifest
	if err := json.Unmarshal(manifestContent, &manifestValue); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifestValue.TexlrVersion != "test" || manifestValue.Engine != "tectonic" {
		t.Fatalf("unexpected manifest: %#v", manifestValue)
	}
}

func TestBuildRequiresForce(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "handoff.tex")
	pdf := filepath.Join(root, "handoff.pdf")
	writeTestFile(t, input, "document")
	writeTestFile(t, pdf, "existing")

	value, err := build(context.Background(), buildOptions{
		inputPath:  input,
		pdfPath:    pdf,
		sourcePath: filepath.Join(root, "source"),
	})
	if err == nil {
		t.Fatal("build unexpectedly succeeded")
	}
	if value.Error == nil || value.Error.Kind != "output_exists" {
		t.Fatalf("unexpected result: %#v", value)
	}
}

func TestForceRejectsUnrelatedSourceDirectory(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "handoff.tex")
	sourceOutput := filepath.Join(root, "important-project")
	writeTestFile(t, input, "document")
	marker := filepath.Join(sourceOutput, "keep.txt")
	writeTestFile(t, marker, "do not delete")

	value, err := build(context.Background(), buildOptions{
		inputPath:  input,
		pdfPath:    filepath.Join(root, "handoff.pdf"),
		sourcePath: sourceOutput,
		force:      true,
	})
	if err == nil || value.Error == nil || value.Error.Kind != "invalid_output" {
		t.Fatalf("unexpected result: %#v, %v", value, err)
	}
	if content, readErr := os.ReadFile(marker); readErr != nil || string(content) != "do not delete" {
		t.Fatalf("unrelated directory was modified: %q, %v", content, readErr)
	}
}

func TestOutputPathsMustBeDisjoint(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "handoff.tex")
	writeTestFile(t, input, "document")
	shared := filepath.Join(root, "output")

	value, err := build(context.Background(), buildOptions{
		inputPath:  input,
		pdfPath:    shared,
		sourcePath: shared,
		force:      true,
	})
	if err == nil || value.Error == nil || value.Error.Kind != "invalid_output" {
		t.Fatalf("unexpected result: %#v, %v", value, err)
	}
}

func TestOutputPathsResolveSymlinkedParents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup requires additional privileges on Windows")
	}
	root := t.TempDir()
	input := filepath.Join(root, "handoff.tex")
	writeTestFile(t, input, "document")
	realDirectory := filepath.Join(root, "real")
	if err := os.Mkdir(realDirectory, 0o755); err != nil {
		t.Fatalf("mkdir real directory: %v", err)
	}
	aliasDirectory := filepath.Join(root, "alias")
	if err := os.Symlink(realDirectory, aliasDirectory); err != nil {
		t.Fatalf("create directory alias: %v", err)
	}

	value, err := build(context.Background(), buildOptions{
		inputPath:  input,
		pdfPath:    filepath.Join(realDirectory, "shared"),
		sourcePath: filepath.Join(root, "source"),
		logPath:    filepath.Join(aliasDirectory, "shared"),
	})
	if err == nil || value.Error == nil || value.Error.Kind != "invalid_output" {
		t.Fatalf("unexpected result: %#v, %v", value, err)
	}
}

func TestPublishPreparedRollsBackEarlierOutputs(t *testing.T) {
	root := t.TempDir()
	firstSource := filepath.Join(root, "first-new")
	secondSource := filepath.Join(root, "second-new")
	firstDestination := filepath.Join(root, "first")
	secondParent := filepath.Join(root, "second-parent")
	secondDestination := filepath.Join(secondParent, "second")
	writeTestFile(t, firstSource, "new first")
	writeTestFile(t, secondSource, "new second")
	writeTestFile(t, firstDestination, "old first")

	first, err := prepareFileOutput(firstSource, firstDestination, true)
	if err != nil {
		t.Fatalf("prepare first: %v", err)
	}
	second, err := prepareFileOutput(secondSource, secondDestination, true)
	if err != nil {
		t.Fatalf("prepare second: %v", err)
	}
	if err := os.RemoveAll(secondParent); err != nil {
		t.Fatalf("remove second parent: %v", err)
	}
	if err := publishPrepared([]preparedOutput{first, second}); err == nil {
		t.Fatal("publish unexpectedly succeeded")
	}
	content, err := os.ReadFile(firstDestination)
	if err != nil || string(content) != "old first" {
		t.Fatalf("first output was not rolled back: %q, %v", content, err)
	}
}

func TestRollbackPreservesBackupWhenRestoreFails(t *testing.T) {
	root := t.TempDir()
	backup := filepath.Join(root, "preserved-backup")
	writeTestFile(t, backup, "old artifact")
	output := preparedOutput{
		destination: filepath.Join(root, "missing-parent", "artifact"),
		backup:      backup,
	}
	if err := rollbackPrepared([]preparedOutput{output}); err == nil {
		t.Fatal("rollback unexpectedly succeeded")
	}
	cleanupPrepared([]preparedOutput{output})
	content, err := os.ReadFile(backup)
	if err != nil || string(content) != "old artifact" {
		t.Fatalf("backup was not preserved: %q, %v", content, err)
	}
}

func TestRunEmitsJSONUsageResult(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"build", "handoff.tex", "--json"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
	var value result
	if err := json.Unmarshal(stdout.Bytes(), &value); err != nil {
		t.Fatalf("decode JSON: %v; output: %s", err, stdout.String())
	}
	if value.Error == nil || value.Error.Kind != "usage_error" {
		t.Fatalf("unexpected result: %#v", value)
	}
}

func TestJSONFailureIncludesRequestedPaths(t *testing.T) {
	root := t.TempDir()
	pdfPath := filepath.Join(root, "handoff.pdf")
	sourcePath := filepath.Join(root, "source")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{
		"build", filepath.Join(root, "missing.tex"),
		"--pdf", pdfPath,
		"--source", sourcePath,
		"--json",
	}, &stdout, &stderr, "test")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	var value result
	if err := json.Unmarshal(stdout.Bytes(), &value); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	expectedPDF, err := canonicalOutputPath(pdfPath)
	if err != nil {
		t.Fatalf("canonical PDF path: %v", err)
	}
	expectedSource, err := canonicalOutputPath(sourcePath)
	if err != nil {
		t.Fatalf("canonical source path: %v", err)
	}
	if value.PDFPath != expectedPDF || value.SourcePath != expectedSource {
		t.Fatalf("requested paths missing: %#v", value)
	}
}

func installFakeTools(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	writeExecutable(t, filepath.Join(directory, "dot"), `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then shift; output="$1"; fi
  shift
done
printf 'fake graphviz pdf' > "$output"
`)
	writeExecutable(t, filepath.Join(directory, "mmdc"), `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then shift; output="$1"; fi
  shift
done
printf 'fake mermaid pdf' > "$output"
`)
	writeExecutable(t, filepath.Join(directory, "gs"), `#!/bin/sh
for argument in "$@"; do
  case "$argument" in
    -sOutputFile=*) output="${argument#-sOutputFile=}" ;;
    *.pdf) input="$argument" ;;
  esac
done
cp "$input" "$output"
`)
	writeExecutable(t, filepath.Join(directory, "tectonic"), `#!/bin/sh
while [ "$#" -gt 0 ]; do
  case "$1" in
    *.tex) input="$1" ;;
    --outdir) shift; output_dir="$1" ;;
  esac
  shift
done
base="${input%.tex}"
mkdir -p "$output_dir"
printf 'fake handoff pdf' > "$output_dir/$base.pdf"
printf 'fake compiler log' > "$output_dir/$base.log"
`)
	return directory
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	writeTestFile(t, path, content)
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
