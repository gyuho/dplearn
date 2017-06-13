// Package fileutil implements basic file utilities.
package fileutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

// ReadDir returns the filenames in the given directory in sorted order.
func ReadDir(dirpath string) ([]string, error) {
	dir, err := os.Open(dirpath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

const (
	// PrivateFileMode grants owner to read/write a file.
	PrivateFileMode = 0600
	// PrivateDirMode grants owner to make/remove files inside the directory.
	PrivateDirMode = 0700
)

// WriteToFile writes data to a file.
func WriteToFile(fpath string, data []byte) error {
	f, err := openToOverwrite(fpath)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return err
		}
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

func openToRead(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDONLY, PrivateFileMode)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func openToOverwrite(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, PrivateFileMode)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func openToAppend(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, PrivateFileMode)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Exist returns true if the file or directorE exists.
func Exist(fpath string) bool {
	st, err := os.Stat(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	if st.IsDir() {
		return true
	}
	if _, err := os.Stat(fpath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

type walkMode int

const (
	fileOnly walkMode = iota
	directoryOnly
	all
)

func getLevel(targetDir, dir string) int {
	return _getLevel(targetDir, dir, 0)
}

func _getLevel(targetDir, dir string, lvl int) int {
	if len(dir) == 0 {
		return -1
	}
	if targetDir == dir {
		return lvl
	}
	return _getLevel(targetDir, filepath.Dir(dir), lvl+1)
}

func walk(targetDir string, mode walkMode) (map[string]os.FileInfo, error) {
	rm := make(map[string]os.FileInfo)
	visit := func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return nil
		}
		ok := false
		switch mode {
		case fileOnly:
			ok = !f.IsDir()
		case directoryOnly:
			ok = f.IsDir()
		case all:
			ok = true
		}
		if !ok {
			return nil
		}
		if filepath.HasPrefix(path, ".") || strings.Contains(path, "/.") {
			return nil
		}
		rm[path] = f
		return nil
	}
	if err := filepath.Walk(targetDir, visit); err != nil {
		return nil, err
	}
	return rm, nil
}

// FileInfo represents a file info.
type FileInfo struct {
	Path    string
	Size    uint64
	SizeTxt string
	Level   int
}

// GetFileInfo returns the file info of a single file.
func GetFileInfo(fpath string) (FileInfo, error) {
	s, err := os.Stat(fpath)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{Path: fpath, Size: uint64(s.Size()), SizeTxt: humanize.Bytes(uint64(s.Size()))}, nil
}

// FileInfoSlice is a slice of FileInfo.
type FileInfoSlice []FileInfo

func (f FileInfoSlice) Len() int      { return len(f) }
func (f FileInfoSlice) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f FileInfoSlice) Less(i, j int) bool {
	return f[i].Level < f[j].Level || (f[i].Level == f[j].Level && f[i].Path < f[j].Path)
}

// WalkDirectories walks the directory and returns the directory file infos.
func WalkDirectories(dir string) ([]FileInfo, error) {
	rm, err := walk(dir, directoryOnly)
	if err != nil {
		return nil, err
	}

	var fs []FileInfo
	for k, v := range rm {
		fv := FileInfo{
			Path:    k,
			Size:    uint64(v.Size()),
			SizeTxt: humanize.Bytes(uint64(v.Size())),
			Level:   getLevel(dir, k),
		}
		fs = append(fs, fv)
	}
	sort.Sort(FileInfoSlice(fs))

	return fs, nil
}

// WalkFiles walks the directory and returns the file infos.
func WalkFiles(dir string) ([]FileInfo, error) {
	rm, err := walk(dir, fileOnly)
	if err != nil {
		return nil, err
	}

	var fs []FileInfo
	for k, v := range rm {
		fv := FileInfo{
			Path:    k,
			Size:    uint64(v.Size()),
			SizeTxt: humanize.Bytes(uint64(v.Size())),
		}
		fs = append(fs, fv)
	}
	sort.Sort(FileInfoSlice(fs))

	return fs, nil
}

// IsDirWriteable checks if dir is writable by writing and removing a file
// to dir. It returns nil if dir is writable.
func IsDirWriteable(dir string) error {
	f := filepath.Join(dir, ".touch")
	if err := ioutil.WriteFile(f, []byte(""), PrivateFileMode); err != nil {
		return err
	}
	return os.Remove(f)
}

// TouchDirAll is similar to os.MkdirAll. It creates directories with 0700 permission if any directory
// does not exists. TouchDirAll also ensures the given directory is writable.
func TouchDirAll(dir string) error {
	// If path is already a directory, MkdirAll does nothing
	// and returns nil.
	err := os.MkdirAll(dir, PrivateDirMode)
	if err != nil {
		// if mkdirAll("a/text") and "text" is not
		// a directory, this will return syscall.ENOTDIR
		return err
	}
	return IsDirWriteable(dir)
}
