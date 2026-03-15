package output

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Writer interface {
	Write(content string) error
	Target() string
}

type StdoutWriter struct {
	writer io.Writer
}

func NewStdoutWriter(writer io.Writer) *StdoutWriter {
	if writer == nil {
		writer = os.Stdout
	}
	return &StdoutWriter{writer: writer}
}

func (w *StdoutWriter) Write(content string) error {
	_, err := io.WriteString(w.writer, content)
	return err
}

func (w *StdoutWriter) Target() string {
	return "stdout"
}

type FileWriter struct {
	path string
}

func NewFileWriter(path string) (*FileWriter, error) {
	if path == "" {
		return nil, fmt.Errorf("output path required")
	}
	return &FileWriter{path: path}, nil
}

func (w *FileWriter) Write(content string) error {
	dir := filepath.Dir(w.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}
	if err := os.WriteFile(w.path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (w *FileWriter) Target() string {
	return w.path
}

func NewWriter(path string) (Writer, error) {
	if path == "" {
		return NewStdoutWriter(nil), nil
	}
	return NewFileWriter(path)
}
