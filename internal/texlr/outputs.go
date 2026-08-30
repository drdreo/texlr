package texlr

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type preparedOutput struct {
	destination string
	temporary   string
	backup      string
	directory   bool
	replace     bool
}

func inspectFileDestination(path string, force bool) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("file output points to an existing non-regular file: %s", path)
	}
	if !force {
		return fmt.Errorf("output already exists (use --force): %s", path)
	}
	return nil
}

func inspectSourceDestination(path string, force bool) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("source output points to an existing non-directory: %s", path)
	}
	if !force {
		return fmt.Errorf("output already exists (use --force): %s", path)
	}
	empty, err := directoryIsEmpty(path)
	if err != nil {
		return err
	}
	if empty {
		return nil
	}
	manifestPath := filepath.Join(path, "texlr-manifest.json")
	if info, err := os.Stat(manifestPath); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to replace a non-empty directory not created by Texlr: %s", path)
	}
	return nil
}

func directoryIsEmpty(path string) (bool, error) {
	directory, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer directory.Close()
	_, err = directory.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func validateOutputLayout(inputPath, pdfPath, sourcePath, logPath string) error {
	outputs := []string{pdfPath, sourcePath}
	if logPath != "" {
		outputs = append(outputs, logPath)
	}
	for i := range outputs {
		for j := i + 1; j < len(outputs); j++ {
			if pathsOverlap(outputs[i], outputs[j]) {
				return fmt.Errorf("output paths must be distinct and disjoint: %s and %s", outputs[i], outputs[j])
			}
		}
	}

	inputRoot := filepath.Dir(inputPath)
	if pathContains(sourcePath, inputRoot) {
		return fmt.Errorf("source output must not contain the input document tree: %s", sourcePath)
	}
	if pdfPath == inputPath || logPath == inputPath {
		return fmt.Errorf("an output path must not replace the input document: %s", inputPath)
	}
	return nil
}

func pathsOverlap(first, second string) bool {
	return pathContains(first, second) || pathContains(second, first)
}

func pathContains(parent, child string) bool {
	parent = comparablePath(parent)
	child = comparablePath(child)
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !startsWithParent(relative)
}

func comparablePath(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func prepareFileOutput(source, destination string, replace bool) (preparedOutput, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return preparedOutput{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".texlr-output-*")
	if err != nil {
		return preparedOutput{}, err
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return preparedOutput{}, err
	}
	info, err := os.Stat(source)
	if err != nil {
		_ = os.Remove(temporaryPath)
		return preparedOutput{}, err
	}
	if err := copyFile(source, temporaryPath, info.Mode().Perm()); err != nil {
		_ = os.Remove(temporaryPath)
		return preparedOutput{}, err
	}
	return preparedOutput{destination: destination, temporary: temporaryPath, replace: replace}, nil
}

func prepareDirectoryOutput(source, destination string, replace bool) (preparedOutput, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return preparedOutput{}, err
	}
	temporary, err := os.MkdirTemp(filepath.Dir(destination), ".texlr-source-*")
	if err != nil {
		return preparedOutput{}, err
	}
	if err := copyTree(source, temporary, nil); err != nil {
		_ = os.RemoveAll(temporary)
		return preparedOutput{}, err
	}
	if err := os.Chmod(temporary, 0o755); err != nil {
		_ = os.RemoveAll(temporary)
		return preparedOutput{}, err
	}
	return preparedOutput{destination: destination, temporary: temporary, directory: true, replace: replace}, nil
}

func publishPrepared(outputs []preparedOutput) error {
	defer cleanupPrepared(outputs)
	committed := 0
	for index := range outputs {
		if err := commitPrepared(&outputs[index]); err != nil {
			if rollbackErr := rollbackPrepared(outputs[:committed]); rollbackErr != nil {
				return fmt.Errorf("%w; rollback failed and backups were preserved: %v", err, rollbackErr)
			}
			return err
		}
		committed++
	}
	for index := range outputs {
		if outputs[index].backup != "" {
			_ = os.RemoveAll(outputs[index].backup)
			outputs[index].backup = ""
		}
	}
	return nil
}

func commitPrepared(output *preparedOutput) error {
	var validationErr error
	if output.directory {
		validationErr = inspectSourceDestination(output.destination, output.replace)
	} else {
		validationErr = inspectFileDestination(output.destination, output.replace)
	}
	if validationErr != nil {
		return validationErr
	}

	if _, err := os.Lstat(output.destination); err == nil {
		backup, backupErr := reserveBackupPath(filepath.Dir(output.destination))
		if backupErr != nil {
			return backupErr
		}
		if renameErr := os.Rename(output.destination, backup); renameErr != nil {
			return renameErr
		}
		output.backup = backup
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(output.temporary, output.destination); err != nil {
		if output.backup == "" {
			return err
		}
		if restoreErr := os.Rename(output.backup, output.destination); restoreErr != nil {
			return fmt.Errorf("%w; could not restore backup %s: %v", err, output.backup, restoreErr)
		}
		output.backup = ""
		return err
	}
	output.temporary = ""
	return nil
}

func rollbackPrepared(outputs []preparedOutput) error {
	var rollbackErrors []error
	for index := len(outputs) - 1; index >= 0; index-- {
		if err := os.RemoveAll(outputs[index].destination); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("remove new output %s: %w", outputs[index].destination, err))
			continue
		}
		if outputs[index].backup != "" {
			if err := os.Rename(outputs[index].backup, outputs[index].destination); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore backup %s: %w", outputs[index].backup, err))
			}
		}
	}
	return errors.Join(rollbackErrors...)
}

func cleanupPrepared(outputs []preparedOutput) {
	for _, output := range outputs {
		if output.temporary != "" {
			_ = os.RemoveAll(output.temporary)
		}
	}
}

func reserveBackupPath(parent string) (string, error) {
	file, err := os.CreateTemp(parent, ".texlr-backup-*")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}
