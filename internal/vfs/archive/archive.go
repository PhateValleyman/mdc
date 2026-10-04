package archive

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v4"

	"github.com/kooler/MiddayCommander/internal/vfs"
)

// FS implements vfs.FS for browsing inside archives (read-only).
type FS struct {
	archivePath string // path to the archive file on disk
	afs         archiver.ArchiveFS
}

// New opens an archive file and returns a read-only VFS.
func New(archivePath string) (*FS, error) {
	// Detect format
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	format, _, err := archiver.Identify(archivePath, f)
	if err != nil {
		return nil, err
	}

	ext, ok := format.(archiver.Archival)
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: archivePath, Err: fs.ErrInvalid}
	}

	afs := archiver.ArchiveFS{
		Path:    archivePath,
		Format:  ext,
		Context: context.Background(),
	}

	return &FS{
		archivePath: archivePath,
		afs:         afs,
	}, nil
}

// ArchivePath returns the path to the archive file.
func (f *FS) ArchivePath() string {
	return f.archivePath
}

func (f *FS) Open(name string) (fs.File, error) {
	return f.afs.Open(cleanPath(name))
}

func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	return f.afs.ReadDir(cleanPath(name))
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	return f.afs.Stat(cleanPath(name))
}

// cleanPath normalizes a path for use with ArchiveFS.
// ArchiveFS expects "." for root, no leading slashes.
func cleanPath(name string) string {
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return "."
	}
	return name
}

// IsArchive returns true if the file at path looks like a supported archive.
func IsArchive(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	format, _, err := archiver.Identify(path, f)
	if err != nil {
		return false
	}
	_, ok := format.(archiver.Archival)
	return ok
}

// ExtractFile copies a single file from the archive to destDir.
func (f *FS) ExtractFile(nameInArchive, destDir string) error {
	src, err := f.afs.Open(cleanPath(nameInArchive))
	if err != nil {
		return err
	}
	defer src.Close()

	destPath := filepath.Join(destDir, filepath.Base(nameInArchive))
	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// Verify interface compliance (read-only).
var _ vfs.FS = (*FS)(nil)
