package texlr

import (
	"encoding/json"
	"fmt"
	"io"
)

type result struct {
	Command    string          `json:"command"`
	Success    bool            `json:"success"`
	PDFPath    string          `json:"pdfPath,omitempty"`
	SourcePath string          `json:"sourcePath,omitempty"`
	LogPath    string          `json:"logPath,omitempty"`
	WorkDir    string          `json:"workDir,omitempty"`
	Diagrams   []diagramResult `json:"diagrams,omitempty"`
	Error      *errorResult    `json:"error,omitempty"`
}

type diagramResult struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Output string `json:"output"`
}

type errorResult struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type operationError struct {
	kind    string
	message string
	cause   error
	result  result
}

func (e *operationError) Error() string {
	if e.cause == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.cause)
}

func writeResult(w io.Writer, value result, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}

	if !value.Success {
		_, err := fmt.Fprintf(w, "texlr: %s\n", value.Error.Message)
		if value.LogPath != "" {
			_, _ = fmt.Fprintf(w, "log: %s\n", value.LogPath)
		}
		if value.WorkDir != "" {
			_, _ = fmt.Fprintf(w, "retained build: %s\n", value.WorkDir)
		}
		return err
	}

	_, err := fmt.Fprintf(w, "texlr: %s succeeded\n", value.Command)
	if value.PDFPath != "" {
		_, _ = fmt.Fprintf(w, "pdf: %s\n", value.PDFPath)
	}
	if value.SourcePath != "" {
		_, _ = fmt.Fprintf(w, "source: %s\n", value.SourcePath)
	}
	if value.LogPath != "" {
		_, _ = fmt.Fprintf(w, "log: %s\n", value.LogPath)
	}
	return err
}
