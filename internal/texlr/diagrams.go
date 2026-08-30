package texlr

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	diagramPattern = regexp.MustCompile(`(?s)\\(graphviz|mermaid)\s*(?:\[[^\]]*\])?\s*\{([^{}]+)\}`)
	includePattern = regexp.MustCompile(`(?s)\\(?:input|include)\s*\{([^{}]+)\}`)
)

var ignoredEnvironments = []string{"verbatim", "verbatim*", "Verbatim", "lstlisting", "minted", "comment"}

type diagram struct {
	kind       string
	sourcePath string
	outputPath string
	relSource  string
	relOutput  string
}

func discoverDiagrams(root, entryPath string) ([]diagram, error) {
	seenDocuments := make(map[string]bool)
	seenDiagrams := make(map[string]bool)
	queue := []string{entryPath}
	var diagrams []diagram

	for len(queue) > 0 {
		path := filepath.Clean(queue[0])
		queue = queue[1:]
		if seenDocuments[path] {
			continue
		}
		seenDocuments[path] = true
		if !pathWithin(root, path) {
			return nil, fmt.Errorf("included TeX file escapes the document root: %s", path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read active TeX file %s: %w", path, err)
		}
		activeContent := sanitizeLaTeX(content)

		for _, match := range diagramPattern.FindAllSubmatch(activeContent, -1) {
			item, err := diagramFromMatch(root, match)
			if err != nil {
				return nil, err
			}
			key := item.kind + "\x00" + item.sourcePath
			if !seenDiagrams[key] {
				seenDiagrams[key] = true
				diagrams = append(diagrams, item)
			}
		}
		for _, match := range includePattern.FindAllSubmatch(activeContent, -1) {
			includePath, err := resolveInclude(root, string(match[1]))
			if err != nil {
				return nil, err
			}
			queue = append(queue, includePath)
		}
	}

	sort.Slice(diagrams, func(i, j int) bool {
		if diagrams[i].kind == diagrams[j].kind {
			return diagrams[i].relSource < diagrams[j].relSource
		}
		return diagrams[i].kind < diagrams[j].kind
	})
	return diagrams, nil
}

func diagramFromMatch(root string, match [][]byte) (diagram, error) {
	kind := string(match[1])
	relative := filepath.Clean(filepath.FromSlash(strings.TrimSpace(string(match[2]))))
	if filepath.IsAbs(relative) || startsWithParent(relative) {
		return diagram{}, fmt.Errorf("%s source must stay inside the document root: %s", kind, relative)
	}
	source := filepath.Join(root, relative)
	if !pathWithin(root, source) {
		return diagram{}, fmt.Errorf("%s source escapes the document root: %s", kind, relative)
	}
	return diagram{
		kind:       kind,
		sourcePath: source,
		outputPath: source + ".pdf",
		relSource:  filepath.ToSlash(relative),
		relOutput:  filepath.ToSlash(relative) + ".pdf",
	}, nil
}

func resolveInclude(root, value string) (string, error) {
	relative := filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if filepath.Ext(relative) == "" {
		relative += ".tex"
	}
	if filepath.IsAbs(relative) || startsWithParent(relative) {
		return "", fmt.Errorf("included TeX file must stay inside the document root: %s", relative)
	}
	path := filepath.Join(root, relative)
	if !pathWithin(root, path) {
		return "", fmt.Errorf("included TeX file escapes the document root: %s", relative)
	}
	return path, nil
}

func sanitizeLaTeX(content []byte) []byte {
	sanitized := stripComments(stripInlineVerbs(append([]byte(nil), content...)))
	for _, environment := range ignoredEnvironments {
		pattern := regexp.MustCompile(`(?s)\\begin\s*\{` + regexp.QuoteMeta(environment) + `\}.*?\\end\s*\{` + regexp.QuoteMeta(environment) + `\}`)
		sanitized = pattern.ReplaceAllFunc(sanitized, blankExceptNewlines)
	}
	return sanitized
}

func stripInlineVerbs(content []byte) []byte {
	for index := 0; index+5 < len(content); index++ {
		if !bytes.HasPrefix(content[index:], []byte(`\verb`)) {
			continue
		}
		delimiterIndex := index + 5
		if delimiterIndex < len(content) && content[delimiterIndex] == '*' {
			delimiterIndex++
		}
		if delimiterIndex >= len(content) || content[delimiterIndex] == '\n' {
			continue
		}
		delimiter := content[delimiterIndex]
		end := bytes.IndexByte(content[delimiterIndex+1:], delimiter)
		if end < 0 {
			continue
		}
		end += delimiterIndex + 2
		for position := index; position < end; position++ {
			if content[position] != '\n' {
				content[position] = ' '
			}
		}
		index = end - 1
	}
	return content
}

func stripComments(content []byte) []byte {
	for index := 0; index < len(content); index++ {
		if content[index] != '%' {
			continue
		}
		backslashes := 0
		for position := index - 1; position >= 0 && content[position] == '\\'; position-- {
			backslashes++
		}
		if backslashes%2 == 1 {
			continue
		}
		for index < len(content) && content[index] != '\n' {
			content[index] = ' '
			index++
		}
	}
	return content
}

func blankExceptNewlines(content []byte) []byte {
	for index := range content {
		if content[index] != '\n' {
			content[index] = ' '
		}
	}
	return content
}

func renderDiagrams(ctx context.Context, diagrams []diagram, log io.Writer) ([]diagramResult, error) {
	results := make([]diagramResult, 0, len(diagrams))
	for _, item := range diagrams {
		info, err := os.Stat(item.sourcePath)
		if err != nil {
			return results, fmt.Errorf("missing %s source %s: %w", item.kind, item.relSource, err)
		}
		if !info.Mode().IsRegular() {
			return results, fmt.Errorf("%s source is not a regular file: %s", item.kind, item.relSource)
		}
		if err := os.MkdirAll(filepath.Dir(item.outputPath), 0o755); err != nil {
			return results, err
		}
		rawFile, err := os.CreateTemp(filepath.Dir(item.outputPath), ".texlr-diagram-*.pdf")
		if err != nil {
			return results, err
		}
		rawPath := rawFile.Name()
		if err := rawFile.Close(); err != nil {
			_ = os.Remove(rawPath)
			return results, err
		}
		var name string
		var args []string
		switch item.kind {
		case "graphviz":
			name = "dot"
			args = []string{"-Tpdf", "-o", rawPath, item.sourcePath}
		case "mermaid":
			name = "mmdc"
			args = []string{"-i", item.sourcePath, "-o", rawPath, "-b", "transparent", "-t", "neutral"}
		default:
			_ = os.Remove(rawPath)
			return results, fmt.Errorf("unsupported diagram kind %q", item.kind)
		}
		if err := execute(ctx, log, "", name, args...); err != nil {
			_ = os.Remove(rawPath)
			return results, fmt.Errorf("render %s diagram %s: %w", item.kind, item.relSource, err)
		}
		normalizeArgs := []string{
			"-q", "-dSAFER", "-dBATCH", "-dNOPAUSE",
			"-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.5",
			"-sOutputFile=" + item.outputPath, rawPath,
		}
		if err := execute(ctx, log, "", "gs", normalizeArgs...); err != nil {
			_ = os.Remove(rawPath)
			_ = os.Remove(item.outputPath)
			return results, fmt.Errorf("normalize %s diagram %s: %w", item.kind, item.relSource, err)
		}
		_ = os.Remove(rawPath)
		results = append(results, diagramResult{Kind: item.kind, Source: item.relSource, Output: item.relOutput})
	}
	return results, nil
}

func execute(ctx context.Context, log io.Writer, directory, name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("required executable %q was not found in PATH", name)
	}
	_, _ = fmt.Fprintf(log, "$ %s %s\n", name, strings.Join(args, " "))
	var command *exec.Cmd
	switch name {
	case "dot":
		command = exec.CommandContext(ctx, "dot", args...)
	case "mmdc":
		command = exec.CommandContext(ctx, "mmdc", args...)
	case "tectonic":
		command = exec.CommandContext(ctx, "tectonic", args...)
	case "gs":
		command = exec.CommandContext(ctx, "gs", args...)
	default:
		return fmt.Errorf("unsupported executable %q", name)
	}
	command.Dir = directory
	command.Env = commandEnvironment(name, log)
	var output bytes.Buffer
	command.Stdout = io.MultiWriter(log, &output)
	command.Stderr = io.MultiWriter(log, &output)
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}

func commandEnvironment(name string, log io.Writer) []string {
	environment := os.Environ()
	if name != "mmdc" || os.Getenv("PUPPETEER_EXECUTABLE_PATH") != "" {
		return environment
	}

	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
	}
	for _, candidate := range candidates {
		path := candidate
		if !filepath.IsAbs(candidate) {
			var err error
			path, err = exec.LookPath(candidate)
			if err != nil {
				continue
			}
		} else if info, err := os.Stat(candidate); err != nil || !info.Mode().IsRegular() {
			continue
		}
		_, _ = fmt.Fprintf(log, "mermaid browser: %s\n", path)
		return append(environment, "PUPPETEER_EXECUTABLE_PATH="+path)
	}
	return environment
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !startsWithParent(relative)
}
