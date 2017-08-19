package main

import (
	"flag"
	"os"
	"path/filepath"

	containerimage "github.com/gyuho/dplearn/container-image"
	"github.com/gyuho/dplearn/pkg/fileutil"

	"github.com/golang/glog"
)

func main() {
	configPath := flag.String("config", "container.yaml", "Specify config file path.")
	flag.Parse()

	cfg, err := containerimage.Read(*configPath)
	if err != nil {
		glog.Fatal(err)
	}

	dir := cfg.DockerfilesBaseDir
	if !fileutil.Exist(dir) {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			glog.Fatal(err)
		}
	}

	app := filepath.Join(dir, "Dockerfile-app")
	if err = fileutil.WriteToFile(app, []byte(cfg.DockerfileApp)); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", app)

	proxy := filepath.Join(dir, "Dockerfile-reverse-proxy")
	if err = fileutil.WriteToFile(proxy, []byte(cfg.DockerfileReverseProxy)); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", proxy)

	r := filepath.Join(dir, "Dockerfile-r")
	if err = fileutil.WriteToFile(r, []byte(cfg.DockerfileR)); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", r)

	python2CPU := filepath.Join(dir, "Dockerfile-python2-cpu")
	if err = fileutil.WriteToFile(python2CPU, []byte(cfg.DockerfilePython2CPU)); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", python2CPU)

	python2GPU := filepath.Join(dir, "Dockerfile-python2-gpu")
	if err = fileutil.WriteToFile(python2GPU, []byte(cfg.DockerfilePython2GPU)); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", python2GPU)

	python3CPU := filepath.Join(dir, "Dockerfile-python3-cpu")
	if err = fileutil.WriteToFile(python3CPU, []byte(cfg.DockerfilePython3CPU)); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", python3CPU)

	python3GPU := filepath.Join(dir, "Dockerfile-python3-gpu")
	if err = fileutil.WriteToFile(python3GPU, []byte(cfg.DockerfilePython3GPU)); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", python3GPU)
}
