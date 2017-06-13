package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/gyuho/deephardway/pkg/fileutil"
	"github.com/mholt/archiver"
)

func main() {
	fpath := flag.String("file", "", "Specify the file path to unarchive.")
	outputDir := flag.String("output-dir", "", "Specify the output directory path.")
	deleteOutputDir := flag.Bool("delete-output-dir", false, "'true' to delete output directory before unarchive.")
	flag.Parse()

	if *deleteOutputDir {
		glog.Infof("deleting %q", *outputDir)
		os.RemoveAll(*outputDir)
		glog.Infof("deleted %q", *outputDir)
	}

	glog.Infof("unarchiving %q", *fpath)
	for _, ff := range archiver.SupportedFormats {
		if !ff.Match(*fpath) {
			continue
		}

		if err := ff.Open(*fpath, *outputDir); err != nil {
			glog.Fatal(err)
		}
		break
	}
	glog.Infof("unarchived %q", *fpath)

	glog.Infof("%q:", *outputDir)
	fis, err := fileutil.WalkDir(*outputDir)
	if err != nil {
		glog.Fatal(err)
	}
	for _, v := range fis {
		fmt.Printf("%q : %s\n", v.Path, v.SizeTxt)
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
