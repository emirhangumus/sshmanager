package storage

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func CreateFileIfNotExists(filePath string, fileMode os.FileMode) error {
	if err := ValidateRegularPath(filePath); err != nil {
		return err
	}
	if _, err := os.Lstat(filePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer f.Close()
	return nil
}

// IsFileEmpty checks if the specified file is empty.
func IsFileEmpty(filePath string) (bool, error) {
	if err := ValidateRegularPath(filePath); err != nil {
		return false, err
	}
	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	return fileInfo.Size() == 0, nil
}

// SecureDelete best-effort overwrites file bytes before removal.
func SecureDelete(path string) error {
	if err := ValidateRegularPath(path); err != nil {
		return err
	}
	expected, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(expected, actual) {
		_ = f.Close()
		return fmt.Errorf("file changed before secure deletion")
	}
	// Bounded best-effort overwrite; never allocate based on attacker file length.
	buf := make([]byte, 32*1024)
	defer clear(buf)
	remaining := actual.Size()
	for remaining > 0 {
		count := int64(len(buf))
		if remaining < count {
			count = remaining
		}
		if _, err := rand.Read(buf[:count]); err != nil {
			_ = f.Close()
			return err
		}
		n, err := f.Write(buf[:count])
		if err != nil {
			_ = f.Close()
			return err
		}
		remaining -= int64(n)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	current, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() || !os.SameFile(expected, current) {
		return fmt.Errorf("file changed before removal")
	}
	return os.Remove(path)
}

func ReadYAMLFile(filePath string, out interface{}) error {
	data, err := ReadFileRegular(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to unmarshal YAML from file %s: %w", filePath, err)
	}
	return nil
}

func WriteYAMLFile(filePath string, data interface{}, fileMode os.FileMode) error {
	dataBytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to YAML: %w", err)
	}

	if err := WriteFileAtomic(filePath, dataBytes, fileMode); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	return nil
}

// WriteFileAtomic writes data to filePath by using a temp file and atomic rename.
func WriteFileAtomic(filePath string, data []byte, fileMode os.FileMode) error {
	if err := ValidateRegularPath(filePath); err != nil {
		return err
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := ValidateRegularPath(filePath); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, "."+filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	shouldCleanup := true
	defer func() {
		if shouldCleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(fileMode); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to set mode on temporary file %s: %w", tmpPath, err)
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temporary file %s: %w", tmpPath, err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to sync temporary file %s: %w", tmpPath, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file %s: %w", tmpPath, err)
	}

	if err := ValidateRegularPath(filePath); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to atomically replace %s: %w", filePath, err)
	}
	shouldCleanup = false

	// Best-effort directory sync for stronger durability guarantees.
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}

	return nil
}

// ValidateRegularPath rejects linked/nonregular state files and an immediately
// linked data directory. Missing files and directories can be created by callers.
// Ancestor directories are trusted; this is not a same-user race defense.
func ValidateRegularPath(path string) error {
	parent, err := os.Lstat(filepath.Dir(path))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !parent.IsDir() {
		return fmt.Errorf("state directory must be a real directory")
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("state path must be a regular file: %s", path)
	}
	return nil
}

// ReadFileRegular bounds allocations and verifies the opened file matches the
// regular directory entry inspected before opening it.
func ReadFileRegular(path string) ([]byte, error) {
	if err := ValidateRegularPath(path); err != nil {
		return nil, err
	}
	expected, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(expected, actual) {
		return nil, fmt.Errorf("state file changed while opening")
	}
	const maxStateBytes = 64 * 1024 * 1024
	if actual.Size() > maxStateBytes {
		return nil, fmt.Errorf("state file exceeds size limit")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxStateBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxStateBytes {
		clear(data)
		return nil, fmt.Errorf("state file exceeds size limit")
	}
	return data, nil
}
