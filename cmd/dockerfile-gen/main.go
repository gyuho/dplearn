package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	configPath := flag.String("config", "dockerfiles/dev-cpu/config.yaml", "Specify config file path.")
	flag.Parse()

	bts, err := ioutil.ReadFile(*configPath)
	if err != nil {
		glog.Fatal(err)
	}
	var cfg configuration
	if err = yaml.Unmarshal(bts, &cfg); err != nil {
		glog.Fatal(err)
	}
	cfg.Updated = nowPST().String()

	switch cfg.Device {
	case "cpu":
		cfg.NVIDIAcuDNN = "# Built for CPU, no need to install 'cuda'"
	case "gpu":
		cfg.NVIDIAcuDNN = `# Tensorflow GPU image already includes https://developer.nvidia.com/cudnn
# https://github.com/fastai/courses/blob/master/setup/install-gpu.sh
# RUN ls /usr/local/cuda/lib64/
# RUN ls /usr/local/cuda/include/`
	}

	buf := new(bytes.Buffer)
	tp := template.Must(template.New("tmplDockerfile").Parse(tmplDockerfile))
	if err = tp.Execute(buf, &cfg); err != nil {
		glog.Fatal(err)
	}
	txt := buf.String()

	for _, fpath := range cfg.DockerfilePaths {
		if !exist(filepath.Dir(fpath)) {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				glog.Fatal(err)
			}
		}
		if err = toFile(txt, fpath); err != nil {
			glog.Fatal(err)
		}
		glog.Infof("wrote %q", fpath)
	}
}

type configuration struct {
	Updated             string
	Device              string `yaml:"device"`
	TensorflowBaseImage string `yaml:"tensorflow-base-image"`
	NVIDIAcuDNN         string

	NVMVersion      string   `yaml:"nvm-version"`
	NodeVersion     string   `yaml:"node-version"`
	GoVersion       string   `yaml:"go-version"`
	DockerfilePaths []string `yaml:"dockerfile-paths"`
}

const tmplDockerfile = `# Last Updated at {{.Updated}}
# This Dockerfile contains everything needed for development and production use.
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile.gpu
# https://gcr.io/tensorflow/tensorflow

##########################
# Base image to build upon
FROM {{.TensorflowBaseImage}}
##########################

##########################
# Update OS
# Configure 'bash' for 'source' commands
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections \
  && rm /bin/sh \
  && ln -s /bin/bash /bin/sh \
  && ls -l $(which bash) \
  && apt-get -y update \
  && apt-get -y install \
  build-essential \
  gcc \
  apt-utils \
  pkg-config \
  software-properties-common \
  apt-transport-https \
  libssl-dev \
  sudo \
  bash \
  bash-completion \
  tar \
  unzip \
  curl \
  wget \
  git \
  nginx \
  libcupti-dev \
  rsync \
  python \
  python-pip \
  python-dev \
  python3-pip \
  && echo "root ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers \
  && apt-get -y clean \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y dist-upgrade \
  && apt-get -y update \
  && apt-get -y upgrade \
  && apt-get -y autoremove \
  && apt-get -y autoclean
##########################

##########################
# Set working directory
ENV HOME_DIR /
WORKDIR ${HOME_DIR}
##########################

##########################
# Install additional Python libraries
ENV HOME /root

RUN pip --no-cache-dir install \
  requests \
  bcolz \
  theano \
  keras==1.2.2

RUN echo $'[global]\n\
device = {{.Device}}\n\
floatX = float32\n\
[cuda]\n\
root = /usr/local/cuda\n'\
> ${HOME}/.theanorc \
  && cat ${HOME}/.theanorc

RUN mkdir -p ${HOME}/.keras \
  && echo $'{\n\
    "image_dim_ordering": "th",\n\
    "epsilon": 1e-07,\n\
    "floatx": "float32",\n\
    "backend": "theano"\n\
}\n'\
> ${HOME}/.keras/keras.json \
  && cat ${HOME}/.keras/keras.json

{{.NVIDIAcuDNN}}

# Configure Jupyter
ADD ./jupyter_notebook_config.py /root/.jupyter/

# Jupyter has issues with being run directly: https://github.com/ipython/ipython/issues/7062
# We just add a little wrapper script.
ADD ./run_jupyter.sh /
##########################

##########################
# Install Go for backend
ENV GOROOT /usr/local/go
ENV GOPATH /gopath
ENV PATH ${GOPATH}/bin:${GOROOT}/bin:$PATH
ENV GO_VERSION {{.GoVersion}}
ENV GO_DOWNLOAD_URL https://storage.googleapis.com/golang
RUN rm -rf ${GOROOT} \
  && curl -s ${GO_DOWNLOAD_URL}/go${GO_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local/ -xz \
  && mkdir -p ${GOPATH}/src ${GOPATH}/bin \
  && go version
##########################

##########################
# Install etcd
ENV ETCD_GIT_PATH github.com/coreos/etcd

RUN mkdir -p ${GOPATH}/src/github.com/coreos \
  && git clone https://github.com/coreos/etcd --branch master ${GOPATH}/src/${ETCD_GIT_PATH}

WORKDIR ${GOPATH}/src/${ETCD_GIT_PATH}

RUN git reset --hard HEAD \
  && ./build \
  && cp ./bin/* /
##########################

##########################
# Clone source code, dependencies
RUN mkdir -p ${GOPATH}/src/github.com/gyuho/deephardway
ADD . ${GOPATH}/src/github.com/gyuho/deephardway

# Symlinks to notebooks notebooks
RUN ln -s /gopath/src/github.com/gyuho/deephardway /git-deep
##########################

##########################
# Compile backend
WORKDIR ${GOPATH}/src/github.com/gyuho/deephardway
RUN go build -o ./backend-web-server -v ./cmd/backend-web-server
##########################

##########################
# Install Angular, NodeJS for frontend
# 'node' needs to be in $PATH for 'yarn start' command
WORKDIR ${GOPATH}/src/github.com/gyuho/deephardway

ENV NVM_DIR /usr/local/nvm
RUN curl https://raw.githubusercontent.com/creationix/nvm/v{{.NVMVersion}}/install.sh | /bin/bash \
  && echo "Running nvm scripts..." \
  && source $NVM_DIR/nvm.sh \
  && nvm ls-remote \
  && nvm install {{.NodeVersion}} \
  && curl https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - \
  && echo "deb http://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list \
  && apt-get -y update && apt-get -y install yarn \
  && rm -rf ./node_modules \
  && yarn install \
  && npm rebuild node-sass \
  && npm install \
  && cp /usr/local/nvm/versions/node/v{{.NodeVersion}}/bin/node /usr/bin/node
##########################

##########################
# Set working directory
ENV HOME_DIR /
WORKDIR ${HOME_DIR}
##########################

##########################
# Backend, do not expose to host
# Just run with frontend, in one container
# EXPOSE 2200

# Frontend
EXPOSE 4200

# TensorBoard
EXPOSE 6006

# IPython
EXPOSE 8888

# Web server
EXPOSE 80
##########################

##########################
# Log installed components
RUN cat /etc/lsb-release >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && uname -a >> /container-version.txt \
  && printf "\n" >> /container-version.txt \
  && echo Python2: $(python -V 2>&1) >> /container-version.txt \
  && echo Python3: $(python3 -V 2>&1) >> /container-version.txt \
  && echo IPython: $(ipython -V 2>&1) >> /container-version.txt \
  && echo Jupyter: $(jupyter --version 2>&1) >> /container-version.txt \
  && echo Go: $(go version 2>&1) >> /container-version.txt \
  && echo yarn: $(yarn --version 2>&1) >> /container-version.txt \
  && echo node: $(node --version 2>&1) >> /container-version.txt \
  && echo NPM: $(/usr/local/nvm/versions/node/v{{.NodeVersion}}/bin/npm --version 2>&1) >> /container-version.txt \
  && echo etcd: $(/etcd --version 2>&1) >> /container-version.txt \
  && echo etcdctl: $(ETCDCTL_API=3 /etcdctl version 2>&1) >> /container-version.txt \
  && cat ${GOPATH}/src/github.com/gyuho/deephardway/git-tensorflow.json >> /container-version.txt \
  && cat ${GOPATH}/src/github.com/gyuho/deephardway/git-fastai-courses.json >> /container-version.txt \
  && cat /container-version.txt
##########################
`

func toFile(txt, fpath string) error {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			glog.Fatal(err)
		}
	}
	defer f.Close()
	if _, err := f.WriteString(txt); err != nil {
		glog.Fatal(err)
	}
	return nil
}

// exist returns true if the file or directory exists.
func exist(fpath string) bool {
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

func nowPST() time.Time {
	tzone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(tzone)
}
