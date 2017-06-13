// Package fileutil implements basic file utilities.
package fileutil

import (
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

func walk(targetDir string) (map[string]os.FileInfo, error) {
	rm := make(map[string]os.FileInfo)
	visit := func(path string, f os.FileInfo, err error) error {
		if f != nil {
			if !f.IsDir() {
				if !filepath.HasPrefix(path, ".") && !strings.Contains(path, "/.") {
					wd, err := os.Getwd()
					if err != nil {
						return err
					}
					rm[filepath.Join(wd, strings.Replace(path, wd, "", -1))] = f
				}
			}
		}
		return nil
	}
	err := filepath.Walk(targetDir, visit)
	if err != nil {
		return nil, err
	}
	return rm, nil
}

// FileInfo represents a file info.
type FileInfo struct {
	Path    string
	Size    uint64
	SizeTxt string
}

// FileInfoSlice is a slice of FileInfo.
type FileInfoSlice []FileInfo

func (f FileInfoSlice) Len() int           { return len(f) }
func (f FileInfoSlice) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f FileInfoSlice) Less(i, j int) bool { return f[i].Size < f[j].Size }

// WalkDir walks the directory and returns the file infos.
func WalkDir(dir string) ([]FileInfo, error) {
	rm, err := walk(dir)
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
