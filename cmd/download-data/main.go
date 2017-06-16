package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gyuho/deephardway/pkg/fileutil"
	"github.com/gyuho/deephardway/pkg/urlutil"

	"github.com/golang/glog"
	"github.com/gyuho/archiver"
)

func main() {
	sourcePath := flag.String("source-path", "", "Specify the URL to download from.")
	targetPath := flag.String("target-path", "", "Specify the file path to store.")
	outputDir := flag.String("output-dir", "", "Specify the output directory to unarchive.")
	outputDirOverwrite := flag.Bool("output-dir-overwrite", false, "'true' to delete output directory before unarchive.")
	smartRename := flag.Bool("smart-rename", false, "'true' to update redundant directory hierarchy.")
	verbose := flag.Bool("verbose", false, "'true' to run with 'verbose' mode.")
	flag.Parse()

	size, sizet, err := urlutil.GetContentLength(*sourcePath)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof("%q size is %s", *sourcePath, sizet)

	needDownload := true
	if fileutil.Exist(*targetPath) {
		glog.Infof("%q exists, comparing the size", *targetPath)
		fi, err := fileutil.GetFileInfo(*targetPath)
		if err != nil {
			glog.Fatal(err)
		}
		if fi.Size == size {
			needDownload = false
			glog.Infof("%q(%s) == %q(%s) (no need to download)", *sourcePath, sizet, *targetPath, fi.SizeTxt)
		} else {
			glog.Warningf("target file %q expected %d, source %q has %d", *targetPath, fi.Size, *sourcePath, size)
		}
	}

	if needDownload {
		glog.Infof("downloading %q to %q", *sourcePath, *targetPath)
		data, err := urlutil.Get(urlutil.TrimQuery(*sourcePath))
		if err != nil {
			glog.Fatal(err)
		}
		if !fileutil.Exist(filepath.Dir(*targetPath)) {
			if err = fileutil.TouchDirAll(filepath.Dir(*targetPath)); err != nil {
				glog.Fatal(err)
			}
		}
		if err = fileutil.WriteToFile(*targetPath, data); err != nil {
			glog.Fatal(err)
		}
		glog.Infof("downloaded %q to %q", *sourcePath, *targetPath)
	}

	var ff archiver.Archiver
	for _, format := range archiver.SupportedFormats {
		if format.Match(*targetPath) {
			ff = format
			break
		}
	}

	if ff != nil {
		parentDir := filepath.Dir(*outputDir)
		if !fileutil.Exist(parentDir) {
			glog.Infof("creating %q", parentDir)
			if err := fileutil.TouchDirAll(parentDir); err != nil {
				glog.Fatal(err)
			}
			glog.Infof("created %q", parentDir)
		}

		if *outputDirOverwrite {
			glog.Infof("deleting %q", *outputDir)
			os.RemoveAll(*outputDir)
			glog.Infof("deleted %q", *outputDir)
		}

		glog.Infof("unarchiving %q", *targetPath)
		var opts []archiver.OpOption
		if *verbose {
			opts = append(opts, archiver.WithVerbose())
		}
		if err := ff.Open(*targetPath, *outputDir, opts...); err != nil {
			glog.Fatal(err)
		}
		glog.Infof("unarchived %q", *targetPath)

		if *smartRename {
			glog.Infof("parent directory: %q (base %s)", parentDir, filepath.Base(parentDir))
			glog.Infof("output directory: %q (base %s)", *outputDir, filepath.Base(*outputDir))
			dirs, err := fileutil.WalkDirectories(*outputDir)
			if err != nil {
				glog.Fatal(err)
			}
			if len(dirs) == 0 {
				glog.Fatalf("got no contents in %q (%v)", *outputDir, dirs)
			}
			lvl1Cnt := 0
			var lvl1 fileutil.FileInfo
			for _, d := range dirs {
				if d.Level == 0 {
					continue
				}
				if d.Level == 1 {
					lvl1Cnt++
					lvl1 = d
				}
			}
			if lvl1Cnt == 1 {
				glog.Infof("found redundancy... cleaning up... %+v", lvl1)

				tmpPath := *outputDir + ".tmp"
				glog.Infof("renaming %q to %q", lvl1.Path, tmpPath)
				if err = os.Rename(lvl1.Path, tmpPath); err != nil {
					glog.Fatal(err)
				}
				glog.Infof("renamed %q to %q", lvl1.Path, tmpPath)

				glog.Infof("removing %q", *outputDir)
				if err = os.RemoveAll(*outputDir); err != nil {
					glog.Fatal(err)
				}
				glog.Infof("removed %q", *outputDir)

				glog.Infof("renaming %q to %q", tmpPath, *outputDir)
				if err = os.Rename(tmpPath, *outputDir); err != nil {
					glog.Fatal(err)
				}
				glog.Infof("renamed %q to %q", tmpPath, *outputDir)

				glog.Infof("updated to %q", *outputDir)
			}
		}
		if *verbose {
			glog.Infof("%q:", *outputDir)
			fis, err := fileutil.WalkFiles(*outputDir)
			if err != nil {
				glog.Fatal(err)
			}
			for _, v := range fis {
				fmt.Printf("%q : %s\n", v.Path, v.SizeTxt)
			}
		}
	} else {
		glog.Infof("no need to unarchive %q", *targetPath)
	}
	glog.Info("success!")
}

/*
import os
import sys
import glob
import zipfile

import requests
import humanize

print('Current working directory:', os.getcwd())
print('Python/System version:', sys.version)

def get_filesize(ep):
    return humanize.naturalsize(requests.head(ep).headers.get('content-length', None))

data_files = glob.glob("data/*")

dogscats_path = 'data/dogscats.zip'
dogscats_url = 'http://files.fast.ai/data/dogscats.zip'
vgg16_path = 'data/vgg16.h5'
vgg16_url = 'http://files.fast.ai/models/vgg16.h5'

if dogscats_path not in data_files:
    print(get_filesize(dogscats_url))
else:
    print(dogscats_path, 'exists')
    zip_ref = zipfile.ZipFile(dogscats_path, 'r')
    zip_ref.extractall('data/')
    zip_ref.close()
    print(dogscats_path, 'unzipped')

if vgg16_path not in data_files:
    print(get_filesize(vgg16_url))
else:
    print(vgg16_path, 'exists')
*/
