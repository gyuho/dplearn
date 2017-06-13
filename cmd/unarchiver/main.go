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
