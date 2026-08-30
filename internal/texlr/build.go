package texlr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type buildOptions struct {
	inputPath    string
	pdfPath      string
	sourcePath   string
	logPath      string
	force        bool
	asJSON       bool
	validateOnly bool
	version      string
}

type manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	TexlrVersion  string          `json:"texlrVersion"`
	BuiltAt       string          `json:"builtAt"`
	Input         string          `json:"input"`
	Engine        string          `json:"engine"`
	Diagrams      []diagramResult `json:"diagrams"`
}

func build(ctx context.Context, options buildOptions) (result, error) {
	value := result{Command: "build"}
	if options.validateOnly {
		value.Command = "validate"
	}

	resolved, err := resolveBuildPaths(options)
	if err != nil {
		return failure(value, "invalid_input", "could not resolve build paths", err)
	}
	options = resolved
	if !options.validateOnly {
		value.PDFPath = options.pdfPath
		value.SourcePath = options.sourcePath
	}
	if options.logPath != "" {
		value.LogPath = options.logPath
	}
	if err := inspectInput(options.inputPath); err != nil {
		return failure(value, "invalid_input", "input is not a readable LaTeX document", err)
	}
	if err := inspectOutputs(options); err != nil {
		kind := "invalid_output"
		if strings.Contains(err.Error(), "already exists") {
			kind = "output_exists"
		}
		return failure(value, kind, err.Error(), nil)
	}

	workDir, err := os.MkdirTemp("", "texlr-build-*")
	if err != nil {
		return failure(value, "staging_failed", "could not create staging directory", err)
	}
	sourceDir := filepath.Join(workDir, "source")
	compileDir := filepath.Join(workDir, "build")
	internalLog := filepath.Join(workDir, "texlr.log")
	if err := os.MkdirAll(compileDir, 0o755); err != nil {
		return failedBuild(value, options, workDir, internalLog, "staging_failed", "could not initialize staging directory", err)
	}
	logFile, err := os.Create(internalLog)
	if err != nil {
		return failedBuild(value, options, workDir, internalLog, "log_failed", "could not create build log", err)
	}

	buildErr := stageAndCompile(ctx, options, sourceDir, compileDir, logFile, &value)
	if closeErr := logFile.Close(); buildErr == nil && closeErr != nil {
		buildErr = fmt.Errorf("close build log: %w", closeErr)
	}
	if buildErr != nil {
		var classified *operationError
		if errors.As(buildErr, &classified) {
			return failedBuild(value, options, workDir, internalLog, classified.kind, classified.message, classified.cause)
		}
		return failedBuild(value, options, workDir, internalLog, "build_failed", "document build failed", buildErr)
	}

	if options.validateOnly {
		if options.logPath != "" {
			if err := copyFileAtomic(internalLog, options.logPath, options.force); err != nil {
				return failedBuild(value, options, workDir, internalLog, "output_failed", "could not write validation log", err)
			}
			value.LogPath = options.logPath
		}
		value.Success = true
		if err := os.RemoveAll(workDir); err != nil {
			return failure(value, "cleanup_failed", "validation succeeded but staging cleanup failed", err)
		}
		return value, nil
	}

	pdfSource := filepath.Join(compileDir, strings.TrimSuffix(filepath.Base(options.inputPath), filepath.Ext(options.inputPath))+".pdf")
	if err := writeManifest(sourceDir, options, value.Diagrams); err != nil {
		return failedBuild(value, options, workDir, internalLog, "bundle_failed", "could not write source manifest", err)
	}
	if err := publishOutputs(sourceDir, pdfSource, internalLog, options); err != nil {
		return failedBuild(value, options, workDir, internalLog, "output_failed", "could not publish build outputs", err)
	}

	value.Success = true
	if err := os.RemoveAll(workDir); err != nil {
		return failure(value, "cleanup_failed", "build succeeded but staging cleanup failed", err)
	}
	return value, nil
}

func resolveBuildPaths(options buildOptions) (buildOptions, error) {
	var err error
	options.inputPath, err = canonicalInputPath(options.inputPath)
	if err != nil {
		return options, err
	}
	if options.pdfPath != "" {
		options.pdfPath, err = canonicalOutputPath(options.pdfPath)
		if err != nil {
			return options, err
		}
	}
	if options.sourcePath != "" {
		options.sourcePath, err = canonicalOutputPath(options.sourcePath)
		if err != nil {
			return options, err
		}
	}
	if options.logPath != "" {
		options.logPath, err = canonicalOutputPath(options.logPath)
		if err != nil {
			return options, err
		}
	}
	return options, nil
}

func canonicalInputPath(value string) (string, error) {
	absolute, err := absolutePath(value)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	return canonicalOutputPath(absolute)
}

func canonicalOutputPath(value string) (string, error) {
	absolute, err := absolutePath(value)
	if err != nil {
		return "", err
	}
	parent, err := resolveExistingPath(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

func resolveExistingPath(path string) (string, error) {
	cursor := filepath.Clean(path)
	var suffix []string
	for {
		resolved, err := filepath.EvalSymlinks(cursor)
		if err == nil {
			parts := append([]string{resolved}, suffix...)
			return filepath.Join(parts...), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(cursor)
		if parent == cursor {
			return "", err
		}
		suffix = append([]string{filepath.Base(cursor)}, suffix...)
		cursor = parent
	}
}

func inspectInput(path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".tex") {
		return fmt.Errorf("input must have a .tex extension: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("input is not a regular file: %s", path)
	}
	return nil
}

func inspectOutputs(options buildOptions) error {
	if !options.validateOnly {
		if err := validateOutputLayout(options.inputPath, options.pdfPath, options.sourcePath, options.logPath); err != nil {
			return err
		}
		if err := inspectFileDestination(options.pdfPath, options.force); err != nil {
			return err
		}
		if err := inspectSourceDestination(options.sourcePath, options.force); err != nil {
			return err
		}
	}
	if options.logPath != "" {
		return inspectFileDestination(options.logPath, options.force)
	}
	return nil
}

func stageAndCompile(ctx context.Context, options buildOptions, sourceDir, compileDir string, log io.Writer, value *result) error {
	root := filepath.Dir(options.inputPath)
	excluded := []string{options.pdfPath, options.sourcePath, options.logPath}
	if err := copyTree(root, sourceDir, excluded); err != nil {
		return &operationError{kind: "staging_failed", message: "could not stage document source", cause: err}
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "texlr.cls"), classTemplate, 0o644); err != nil {
		return &operationError{kind: "template_failed", message: "could not install the Texlr class", cause: err}
	}

	diagrams, err := discoverDiagrams(sourceDir, filepath.Join(sourceDir, filepath.Base(options.inputPath)))
	if err != nil {
		return &operationError{kind: "asset_failed", message: "could not inspect diagram references", cause: err}
	}
	value.Diagrams, err = renderDiagrams(ctx, diagrams, log)
	if err != nil {
		return &operationError{kind: "diagram_failed", message: "diagram rendering failed", cause: err}
	}

	inputName := filepath.Base(options.inputPath)
	if err := execute(ctx, log, sourceDir, "tectonic", "-X", "compile", inputName, "--outdir", compileDir, "--keep-logs"); err != nil {
		appendCompilerLog(log, compileDir, inputName)
		return &operationError{kind: "latex_failed", message: "LaTeX compilation failed", cause: err}
	}
	appendCompilerLog(log, compileDir, inputName)
	pdfPath := filepath.Join(compileDir, strings.TrimSuffix(inputName, filepath.Ext(inputName))+".pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		return &operationError{kind: "latex_failed", message: "compiler did not produce the expected PDF", cause: err}
	}
	return nil
}

func appendCompilerLog(destination io.Writer, compileDir, inputName string) {
	name := strings.TrimSuffix(inputName, filepath.Ext(inputName)) + ".log"
	content, err := os.ReadFile(filepath.Join(compileDir, name))
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(destination, "\n--- %s ---\n", name)
	_, _ = destination.Write(content)
}

func writeManifest(sourceDir string, options buildOptions, diagrams []diagramResult) error {
	value := manifest{
		SchemaVersion: 1,
		TexlrVersion:  options.version,
		BuiltAt:       time.Now().UTC().Format(time.RFC3339),
		Input:         filepath.Base(options.inputPath),
		Engine:        "tectonic",
		Diagrams:      diagrams,
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(filepath.Join(sourceDir, "texlr-manifest.json"), content, 0o644)
}

func publishOutputs(sourceDir, pdfSource, internalLog string, options buildOptions) error {
	outputs := make([]preparedOutput, 0, 3)
	defer func() { cleanupPrepared(outputs) }()

	sourceOutput, err := prepareDirectoryOutput(sourceDir, options.sourcePath, options.force)
	if err != nil {
		return fmt.Errorf("prepare source bundle: %w", err)
	}
	outputs = append(outputs, sourceOutput)
	pdfOutput, err := prepareFileOutput(pdfSource, options.pdfPath, options.force)
	if err != nil {
		return fmt.Errorf("prepare PDF: %w", err)
	}
	outputs = append(outputs, pdfOutput)
	if options.logPath != "" {
		logOutput, err := prepareFileOutput(internalLog, options.logPath, options.force)
		if err != nil {
			return fmt.Errorf("prepare log: %w", err)
		}
		outputs = append(outputs, logOutput)
	}
	if err := publishPrepared(outputs); err != nil {
		return fmt.Errorf("commit outputs: %w", err)
	}
	return nil
}

func failedBuild(value result, options buildOptions, workDir, internalLog, kind, message string, cause error) (result, error) {
	value.WorkDir = workDir
	value.LogPath = internalLog
	if options.logPath != "" {
		if err := copyFileAtomic(internalLog, options.logPath, options.force); err == nil {
			value.LogPath = options.logPath
		}
	}
	return failure(value, kind, message, cause)
}

func failure(value result, kind, message string, cause error) (result, error) {
	operation := &operationError{kind: kind, message: message, cause: cause, result: value}
	value.Success = false
	value.Error = &errorResult{Kind: kind, Message: operation.Error()}
	operation.result = value
	return value, operation
}
