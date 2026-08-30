package texlr

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func copyTree(source, destination string, excluded []string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != source && entry.Name() == ".git" && entry.IsDir() {
			return filepath.SkipDir
		}
		if isExcluded(path, excluded) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return copySymlinkTarget(path, target)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func isExcluded(path string, excluded []string) bool {
	clean := filepath.Clean(path)
	for _, candidate := range excluded {
		candidate = filepath.Clean(candidate)
		if clean == candidate {
			return true
		}
		if relative, err := filepath.Rel(candidate, clean); err == nil && relative != "." && relative != ".." && !startsWithParent(relative) {
			return true
		}
	}
	return false
}

func startsWithParent(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == ".."+string(filepath.Separator)
}

func copySymlinkTarget(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("resolve symlink %s: %w", source, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("symlinked directories and special files are not supported: %s", source)
	}
	return copyFile(source, destination, info.Mode().Perm())
}

func copyFile(source, destination string, mode fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chmod(destination, mode)
}

func copyFileAtomic(source, destination string, force bool) error {
	if err := inspectFileDestination(destination, force); err != nil {
		return err
	}
	prepared, err := prepareFileOutput(source, destination, force)
	if err != nil {
		return err
	}
	return publishPrepared([]preparedOutput{prepared})
}
