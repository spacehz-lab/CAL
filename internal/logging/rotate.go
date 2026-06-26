package logging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type rotatingWriter struct {
	path     string
	maxBytes int64
	maxFiles int
	mu       sync.Mutex
}

func newRotatingWriter(path string, maxBytes int64, maxFiles int) (*rotatingWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &rotatingWriter{path: path, maxBytes: maxBytes, maxFiles: maxFiles}, nil
}

func (writer *rotatingWriter) Write(p []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if err := writer.rotateIfNeeded(int64(len(p))); err != nil {
		return 0, err
	}
	file, err := os.OpenFile(writer.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.Write(p)
}

func (writer *rotatingWriter) rotateIfNeeded(incoming int64) error {
	if writer.maxBytes <= 0 {
		return nil
	}
	info, err := os.Stat(writer.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size()+incoming <= writer.maxBytes {
		return nil
	}
	return writer.rotate()
}

func (writer *rotatingWriter) rotate() error {
	if writer.maxFiles <= 0 {
		return os.Remove(writer.path)
	}
	if err := removeIfExists(rotatedPath(writer.path, writer.maxFiles)); err != nil {
		return err
	}
	for index := writer.maxFiles - 1; index >= 1; index-- {
		from := rotatedPath(writer.path, index)
		to := rotatedPath(writer.path, index+1)
		if err := renameIfExists(from, to); err != nil {
			return err
		}
	}
	return renameIfExists(writer.path, rotatedPath(writer.path, 1))
}

func rotatedPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func renameIfExists(from, to string) error {
	if err := os.Rename(from, to); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
