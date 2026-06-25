package deploy

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// createTarGz archives the named paths under baseDir into w.
// Returns the number of compressed bytes written.
func createTarGz(w io.Writer, baseDir string, paths []string) (int64, error) {
	cw := &countWriter{w: w}
	gz := gzip.NewWriter(cw)
	tw := tar.NewWriter(gz)

	for _, p := range paths {
		if err := addPath(tw, baseDir, p); err != nil {
			return 0, err
		}
	}
	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gz.Close(); err != nil {
		return 0, err
	}
	return cw.n, nil
}

// addPath adds a single file or directory tree to tw.
func addPath(tw *tar.Writer, baseDir, relPath string) error {
	src := filepath.Join(baseDir, relPath)
	info, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil // missing paths are silently skipped per Save contract
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return addDirToTar(tw, src, relPath)
	}
	return addFileToTar(tw, src, relPath)
}

func addDirToTar(tw *tar.Writer, srcDir, relBase string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		tarName := filepath.Join(relBase, rel)
		if info.IsDir() {
			return tw.WriteHeader(&tar.Header{
				Name:     tarName + "/",
				Typeflag: tar.TypeDir,
				Mode:     int64(info.Mode()),
				ModTime:  info.ModTime(),
			})
		}
		return addFileToTar(tw, path, tarName)
	})
}

func addFileToTar(tw *tar.Writer, src, name string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:    name,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

// extractTarGz decompresses r into dstDir, recreating the directory structure.
func extractTarGz(r io.Reader, dstDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := extractEntry(tr, hdr, dstDir); err != nil {
			return err
		}
	}
	return nil
}

// extractEntry writes one tar entry into dstDir, guarding against path traversal.
func extractEntry(tr *tar.Reader, hdr *tar.Header, dstDir string) error {
	target := filepath.Join(dstDir, filepath.Clean(hdr.Name))
	if !isUnderDir(target, dstDir) {
		return fmt.Errorf("invalid tar path: %s", hdr.Name)
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, os.FileMode(hdr.Mode))
	case tar.TypeReg:
		return writeExtractedFile(target, hdr, tr)
	}
	return nil
}

func writeExtractedFile(target string, hdr *tar.Header, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// isUnderDir reports whether path is within (or equal to) parent.
func isUnderDir(path, parent string) bool {
	clean := filepath.Clean(parent)
	return strings.HasPrefix(path, clean+string(os.PathSeparator)) || path == clean
}

type countWriter struct {
	w io.Writer
	n int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
