// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TarGz zips source file or directory to dst.tar.gz.
func TarGz(src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gzWriter := gzip.NewWriter(out)
	defer gzWriter.Close()

	tw := tar.NewWriter(gzWriter)
	defer tw.Close()

	return tarFile(tw, src, dst)
}

func tarFile(tw *tar.Writer, src, dst string) error {
	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	var srcBase string
	if si.IsDir() {
		srcBase = filepath.Base(src)
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		hdr, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}
		if srcBase != "" {
			hdr.Name = filepath.Join(srcBase, strings.TrimPrefix(path, src))
		}
		if hdr.Name == dst {
			return nil
		}

		if info.IsDir() {
			hdr.Name += "/"
		}
		if err = tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if hdr.Typeflag == tar.TypeReg {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err = io.CopyN(tw, f, info.Size()); err != nil && err != io.EOF {
				return err
			}
		}
		return nil
	})
}
