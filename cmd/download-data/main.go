package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gyuho/deephardway/pkg/fileutil"
	"github.com/gyuho/deephardway/pkg/urlutil"

	"github.com/golang/glog"
	"github.com/mholt/archiver"
)

func main() {
	sourcePath := flag.String("source-path", "", "Specify the URL to download from.")
	targetPath := flag.String("target-path", "", "Specify the file path to store.")
	outputDir := flag.String("output-dir", "", "Specify the output directory to unarchive.")
	outputDirOverwrite := flag.Bool("output-dir-overwrite", false, "'true' to delete output directory before unarchive.")
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
		if !fileutil.Exist(*outputDir) {
			glog.Infof("creating %q", *outputDir)
			if err := fileutil.TouchDirAll(*outputDir); err != nil {
				glog.Fatal(err)
			}
			glog.Infof("created %q", *outputDir)
		}

		if *outputDirOverwrite {
			glog.Infof("deleting %q", *outputDir)
			os.RemoveAll(*outputDir)
			glog.Infof("deleted %q", *outputDir)
		}

		glog.Infof("unarchiving %q", *targetPath)
		if err := ff.Open(*targetPath, *outputDir); err != nil {
			glog.Fatal(err)
		}
		glog.Infof("unarchived %q", *targetPath)

		glog.Infof("%q:", *outputDir)
		fis, err := fileutil.WalkDir(*outputDir)
		if err != nil {
			glog.Fatal(err)
		}
		for _, v := range fis {
			fmt.Printf("%q : %s\n", v.Path, v.SizeTxt)
		}
	} else {
		glog.Infof("%q cannot be unarchived", *targetPath)
	}
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
