package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vibe-menu/internal/realize/dag"
)

// Writer writes generated files into a base directory.
type Writer struct {
	baseDir string
}

// New returns a Writer that writes to baseDir.
func New(baseDir string) (*Writer, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("create output dir %s: %w", abs, err)
	}
	return &Writer{baseDir: abs}, nil
}

// BaseDir returns the absolute base directory.
func (w *Writer) BaseDir() string { return w.baseDir }

// Write creates baseDir/path with content, making intermediate dirs.
func (w *Writer) Write(path, content string) error {
	full := filepath.Join(w.baseDir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// WriteAllTo writes files into dir (which need not be baseDir).
func (w *Writer) WriteAllTo(dir string, files []dag.GeneratedFile) error {
	tw := &Writer{baseDir: dir}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create temp dir %s: %w", dir, err)
	}
	for _, f := range files {
		if err := tw.Write(f.Path, f.Content); err != nil {
			return err
		}
	}
	return nil
}

// CommitWithPrefix is like Commit but prepends prefix to every destination path.
// This allows the runner to place component files in their subdirectory (e.g.
// "backend", "frontend") while keeping the temp-dir verification prefix-free.
// If prefix is empty or ".", it behaves identically to Commit.
func (w *Writer) CommitWithPrefix(srcDir, prefix string, files []dag.GeneratedFile) error {
	for _, f := range files {
		src := filepath.Join(srcDir, filepath.FromSlash(f.Path))
		dstPath := f.Path
		if prefix != "" && prefix != "." {
			dstPath = filepath.Join(prefix, f.Path)
		}
		dst := filepath.Join(w.baseDir, filepath.FromSlash(dstPath))
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", dstPath, err)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read temp file %s: %w", f.Path, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("commit %s: %w", dstPath, err)
		}
	}
	return nil
}

// Commit moves files from srcDir into baseDir, overwriting existing files.
func (w *Writer) Commit(srcDir string, files []dag.GeneratedFile) error {
	for _, f := range files {
		src := filepath.Join(srcDir, filepath.FromSlash(f.Path))
		dst := filepath.Join(w.baseDir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.Path, err)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read temp file %s: %w", f.Path, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("commit %s: %w", f.Path, err)
		}
	}
	return nil
}
