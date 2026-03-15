package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStdoutWriterWrite(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStdoutWriter(&buf)

	if err := writer.Write("weekly report"); err != nil {
		t.Fatal(err)
	}

	if got := buf.String(); got != "weekly report" {
		t.Fatalf("stdout content = %q, want %q", got, "weekly report")
	}
}

func TestFileWriterWriteCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reports", "weekly.md")
	writer, err := NewFileWriter(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.Write("# Weekly Report\n"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := string(data); got != "# Weekly Report\n" {
		t.Fatalf("file content = %q, want %q", got, "# Weekly Report\n")
	}
}

func TestNewWriterEmptyPathUsesStdout(t *testing.T) {
	writer, err := NewWriter("")
	if err != nil {
		t.Fatal(err)
	}

	if got := writer.Target(); got != "stdout" {
		t.Fatalf("target = %q, want %q", got, "stdout")
	}
}
