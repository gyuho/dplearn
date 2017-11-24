package containerimage

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Config defines Dockerfile template.
type Config struct {
	Updated string

	GoVersion   string `yaml:"go-version"`
	NVMVersion  string `yaml:"nvm-version"`
	NodeVersion string `yaml:"node-version"`

	TensorflowBaseImage string `yaml:"tensorflow-base-image"`
	NVIDIAcuDNN         string
	DockerfilesBaseDir  string `yaml:"dockerfiles-base-dir"`

	KerasVersion string `yaml:"keras-version"`

	DockerfileApp          string
	DockerfileReverseProxy string

	DockerfileR string

	DockerfilePython3CPU string
	DockerfilePython3GPU string
}

// Read reads container image configuration.
func Read(p string) (Config, error) {
	bts, err := ioutil.ReadFile(p)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err = yaml.Unmarshal(bts, &cfg); err != nil {
		return Config{}, err
	}
	cfg.Updated = nowPST().String()

	buf := new(bytes.Buffer)

	if err = template.Must(template.New("tmpl").
		Parse(dockerfileApp)).
		Execute(buf, struct {
			Updated     string
			GoVersion   string
			NVMVersion  string
			NodeVersion string
		}{
			cfg.Updated,
			cfg.GoVersion,
			cfg.NVMVersion,
			cfg.NodeVersion,
		}); err != nil {
		return Config{}, err
	}
	cfg.DockerfileApp = buf.String()
	buf.Reset()

	if err = template.Must(template.New("tmpl").
		Parse(dockerfileReverseProxy)).
		Execute(buf, struct {
			Updated   string
			GoVersion string
		}{
			cfg.Updated,
			cfg.GoVersion,
		}); err != nil {
		return Config{}, err
	}
	cfg.DockerfileReverseProxy = buf.String()
	buf.Reset()

	if err = template.Must(template.New("tmpl").
		Parse(dockerfileR)).
		Execute(buf, struct {
			Updated string
		}{
			cfg.Updated,
		}); err != nil {
		return Config{}, err
	}
	cfg.DockerfileR = buf.String()
	buf.Reset()

	if err = template.Must(template.New("tmpl").
		Parse(dockerfilePython)).
		Execute(buf, struct {
			Updated             string
			Device              string
			TensorflowBaseImage string
			NVIDIAcuDNN         string
			PipCommand          string
			KerasVersion        string
		}{
			cfg.Updated,
			"cpu",
			cfg.TensorflowBaseImage + "-py3",
			"# built for CPU, no need to install cuda",
			"pip3",
			cfg.KerasVersion,
		}); err != nil {
		return Config{}, err
	}
	cfg.DockerfilePython3CPU = buf.String()
	buf.Reset()

	if err = template.Must(template.New("tmpl").
		Parse(dockerfilePython)).
		Execute(buf, struct {
			Updated             string
			Device              string
			TensorflowBaseImage string
			NVIDIAcuDNN         string
			PipCommand          string
			KerasVersion        string
		}{
			cfg.Updated,
			"gpu",
			cfg.TensorflowBaseImage + "-gpu-py3",
			`# Tensorflow GPU image already includes https://developer.nvidia.com/cudnn
# https://github.com/fastai/courses/blob/master/setup/install-gpu.sh
# RUN ls /usr/local/cuda/lib64/
# RUN ls /usr/local/cuda/include/`,
			"pip3",
			cfg.KerasVersion,
		}); err != nil {
		return Config{}, err
	}
	cfg.DockerfilePython3GPU = buf.String()
	buf.Reset()

	return cfg, nil
}

func nowPST() time.Time {
	tzone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(tzone)
}

// TODO(gyuho): separate backend, frontend images
// currently docker doesn't work with --net=host on Mac
// which is my development machine
const dockerfileApp = `##########################
# Last updated at {{.Updated}}
FROM ubuntu:17.10
##########################

##########################
# Set working directory
ENV ROOT_DIR /
WORKDIR ${ROOT_DIR}
ENV HOME /root
##########################

##########################
# Update OS
# Configure 'bash' for 'source' commands
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections \
  && apt-get -y update \
  && apt-get -y install \
  build-essential \
  gcc \
  apt-utils \
  software-properties-common \
  curl \
  python \
  git \
  && rm /bin/sh \
  && ln -s /bin/bash /bin/sh \
  && ls -l $(which bash) \
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
# install Go
ENV GOROOT /usr/local/go
ENV GOPATH /gopath
ENV PATH ${GOPATH}/bin:${GOROOT}/bin:${PATH}
ENV GO_VERSION {{.GoVersion}}
ENV GO_DOWNLOAD_URL https://storage.googleapis.com/golang
RUN rm -rf ${GOROOT} \
  && curl -s ${GO_DOWNLOAD_URL}/go${GO_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local/ -xz \
  && mkdir -p ${GOPATH}/src ${GOPATH}/bin \
  && go version
##########################

##########################
# Clone source code, static assets
# start at repository root
RUN mkdir -p ${GOPATH}/src/github.com/gyuho/dplearn
WORKDIR ${GOPATH}/src/github.com/gyuho/dplearn

ADD ./cmd ${GOPATH}/src/github.com/gyuho/dplearn/cmd
ADD ./backend ${GOPATH}/src/github.com/gyuho/dplearn/backend
ADD ./pkg ${GOPATH}/src/github.com/gyuho/dplearn/pkg
ADD ./vendor ${GOPATH}/src/github.com/gyuho/dplearn/vendor
ADD ./Gopkg.lock ${GOPATH}/src/github.com/gyuho/dplearn/Gopkg.lock
ADD ./Gopkg.toml ${GOPATH}/src/github.com/gyuho/dplearn/Gopkg.toml

ADD ./frontend ${GOPATH}/src/github.com/gyuho/dplearn/frontend
ADD ./angular-cli.json ${GOPATH}/src/github.com/gyuho/dplearn/angular-cli.json
ADD ./package.json ${GOPATH}/src/github.com/gyuho/dplearn/package.json
ADD ./proxy.config.json ${GOPATH}/src/github.com/gyuho/dplearn/proxy.config.json
ADD ./yarn.lock ${GOPATH}/src/github.com/gyuho/dplearn/yarn.lock

ADD ./scripts/docker/run ${GOPATH}/src/github.com/gyuho/dplearn/scripts/docker/run
ADD ./scripts/tests ${GOPATH}/src/github.com/gyuho/dplearn/scripts/tests

RUN go install -v ./cmd/backend-web-server \
  && go install -v ./cmd/gen-frontend-dep
##########################

##########################
# install Angular, NodeJS for frontend
# 'node' needs to be in $PATH for 'yarn start' command
ENV NVM_DIR /usr/local/nvm
RUN pushd ${GOPATH}/src/github.com/gyuho/dplearn \
  && curl https://raw.githubusercontent.com/creationix/nvm/v{{.NVMVersion}}/install.sh | /bin/bash \
  && echo "Running nvm scripts..." \
  && source $NVM_DIR/nvm.sh \
  && nvm ls-remote \
  && nvm install v{{.NodeVersion}} \
  && curl https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - \
  && echo "deb http://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list \
  && apt-get -y update && apt-get -y install yarn \
  && echo "Updating frontend dependencies..." \
  && rm -rf ./node_modules \
  && yarn install \
  && npm rebuild node-sass --force \
  && npm install \
  && nvm alias default {{.NodeVersion}} \
  && nvm alias default node \
  && which node \
  && node -v \
  && cp /usr/local/nvm/versions/node/v{{.NodeVersion}}/bin/node /usr/bin/node \
  && popd
##########################

`

const dockerfileReverseProxy = `##########################
# Last updated at {{.Updated}}
# https://hub.docker.com/_/nginx/
FROM nginx:alpine
##########################

##########################
# Set working directory
ENV ROOT_DIR /
WORKDIR ${ROOT_DIR}
ENV HOME /root
##########################

##########################
RUN set -ex \
  && apk update \
  && apk add --no-cache \
  bash \
  ca-certificates \
  gcc \
  musl-dev \
  openssl \
  curl \
  wget \
  tar \
  git
##########################

##########################
# install Go
ENV GOROOT /usr/local/go
ENV GOPATH /gopath
ENV PATH ${GOPATH}/bin:${GOROOT}/bin:${PATH}
ENV GO_VERSION {{.GoVersion}}
ENV GO_DOWNLOAD_URL https://storage.googleapis.com/golang
RUN rm -rf ${GOROOT} \
  && mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 \
  && curl -s ${GO_DOWNLOAD_URL}/go${GO_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local/ -xz \
  && mkdir -p ${GOPATH}/src ${GOPATH}/bin \
  && go version
##########################

##########################
# Clone source code, static assets
RUN mkdir -p ${GOPATH}/src/github.com/gyuho/dplearn
WORKDIR ${GOPATH}/src/github.com/gyuho/dplearn

ADD ./cmd ${GOPATH}/src/github.com/gyuho/dplearn/cmd
ADD ./backend ${GOPATH}/src/github.com/gyuho/dplearn/backend
ADD ./pkg ${GOPATH}/src/github.com/gyuho/dplearn/pkg
ADD ./vendor ${GOPATH}/src/github.com/gyuho/dplearn/vendor
ADD ./scripts/docker/run ${GOPATH}/src/github.com/gyuho/dplearn/scripts/docker/run

RUN go install -v ./cmd/gen-nginx-conf
##########################

##########################
# Configure reverse proxy
RUN mkdir -p /etc/nginx/sites-available/
ADD nginx.conf /etc/nginx/sites-available/default
##########################

`

/*
# install pandoc, latex for R
# http://pandoc.org/installing.html
# https://github.com/jgm/pandoc/releases
RUN apt-get -y install \
  vim \
  texlive \
  texlive-xetex \
  && wget https://github.com/jgm/pandoc/releases/download/1.19.2.1/pandoc-1.19.2.1-1-amd64.deb \
  && dpkg -i pandoc-1.19.2.1-1-amd64.deb
*/
const dockerfileR = `##########################
# Last updated at {{.Updated}}
FROM ubuntu:17.10
##########################

##########################
# Set working directory
ENV ROOT_DIR /
WORKDIR ${ROOT_DIR}
ENV HOME /root
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
  sudo \
  bash \
  tar \
  unzip \
  curl \
  wget \
  python \
  python-pip \
  python-dev \
  r-base \
  fonts-dejavu \
  gfortran \
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
# install iPython notebook
RUN pip --no-cache-dir install \
  ipykernel \
  jupyter \
  && python -m ipykernel.kernelspec
##########################

##########################
# install R
RUN wget http://repo.continuum.io/miniconda/Miniconda3-3.7.0-Linux-x86_64.sh -O /root/miniconda.sh \
  && bash /root/miniconda.sh -b -p /root/miniconda

# do not overwrite default '/usr/bin/python'
ENV PATH ${PATH}:/root/miniconda/bin

# https://github.com/jupyter/docker-stacks/blob/master/r-notebook/Dockerfile
RUN conda update conda \
  && conda create --yes --name r \
  python=2.7 \
  ipykernel \
  r \
  r-essentials \
  'r-base=3.3.2' \
  'r-irkernel=0.7*' \
  'r-plyr=1.8*' \
  'r-devtools=1.12*' \
  'r-tidyverse=1.0*' \
  'r-shiny=0.14*' \
  'r-rmarkdown=1.2*' \
  'r-forecast=7.3*' \
  'r-rsqlite=1.1*' \
  'r-reshape2=1.4*' \
  'r-nycflights13=0.2*' \
  'r-caret=6.0*' \
  'r-rcurl=1.95*' \
  'r-crayon=1.3*' \
  'r-randomforest=4.6*' \
  && conda clean -tipsy \
  && conda list \
  && source activate r
##########################

##########################
# Configure Jupyter
ADD ./jupyter_notebook_config.py /root/.jupyter/

# Jupyter has issues with being run directly: https://github.com/ipython/ipython/issues/7062
# We just add a little wrapper script.
ADD ./run_jupyter.sh /
##########################

`

/*
https://github.com/fchollet/keras/releases

Keras 2.0.5> "image_data_format": "channels_last"
Keras 1.2.2 "backend": "theano", "image_dim_ordering": "th",
Keras 1.2.2 "backend": "tensorflow", "image_dim_ordering": "tf",
*/
const dockerfilePython = `##########################
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile
# https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/docker/Dockerfile.gpu
# https://gcr.io/tensorflow/tensorflow
FROM {{.TensorflowBaseImage}}
##########################

##########################
# Set working directory
ENV ROOT_DIR /
WORKDIR ${ROOT_DIR}
ENV HOME /root
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
  libcupti-dev \
  rsync \
  python \
  python-pip \
  python-dev \
  python3-pip \
  libhdf5-dev \
  python-tk \
  python3-tk \
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
{{.NVIDIAcuDNN}}
##########################

##########################
# install basic packages
RUN {{.PipCommand}} --no-cache-dir install \
  requests \
  glog \
  humanize \
  bcolz \
  h5py
##########################

##########################
# install Keras
RUN {{.PipCommand}} --no-cache-dir install \
  theano \
  keras=={{.KerasVersion}} \
  && echo $'[global]\n\
device = {{.Device}}\n\
floatX = float32\n\
[cuda]\n\
root = /usr/local/cuda\n'\
> ${HOME}/.theanorc \
  && cat ${HOME}/.theanorc \
  && mkdir -p ${HOME}/.keras/datasets \
  && mkdir -p ${HOME}/.keras/models \
  && echo $'{\n\
  "image_data_format": "channels_last",\n\
  "epsilon": 1e-07,\n\
  "floatx": "float32",\n\
  "backend": "tensorflow"\n\
}\n'\
> ${HOME}/.keras/keras.json \
  && cat ${HOME}/.keras/keras.json
##########################

##########################
# Clone source code, static assets
ADD ./datasets/parameters-cats.npy /root/datasets/parameters-cats.npy
ADD ./backend ${ROOT_DIR}/backend
ADD ./scripts/docker/run ${ROOT_DIR}/scripts/docker/run
ADD ./scripts/tests ${ROOT_DIR}/scripts/tests
##########################

##########################
# Configure Jupyter
ADD ./jupyter_notebook_config.py /root/.jupyter/

# Jupyter has issues with being run directly: https://github.com/ipython/ipython/issues/7062
# We just add a little wrapper script.
ADD ./run_jupyter.sh /
##########################

`
