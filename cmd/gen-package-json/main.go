package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
)

func getGCPPublicIP() (string, error) {
	const endpoint = "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip"
	// curl -L http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip -H 'Metadata-Flavor:Google'
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header = map[string][]string{
		"Metadata-Flavor": []string{"Google"},
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bts)), nil
}

func main() {
	configPath := flag.String("config", "package.json", "Specify config file path.")
	flag.Parse()

	// TODO: angular-cli does not work with public IP, so need to use 0.0.0.0
	// https://github.com/angular/angular-cli/issues/2587#issuecomment-252586913
	// but webpack prod mode does not work with 0.0.0.0
	// so just use without prod mode
	// https://github.com/webpack/webpack-dev-server/issues/882
	cfg := configuration{
		NgServeStartCommand:     "ng serve --prod",
		NgServeStartProdCommand: "ng serve",
		Host: "0.0.0.0",
	}

	// inspect metadata to get public IP
	for i := 0; i < 3; i++ {
		host, err := getGCPPublicIP()
		if err != nil {
			glog.Warning(err)
			time.Sleep(300 * time.Millisecond)
			continue
		}
		cfg.Host = host
		glog.Infof("use public host IP %q", host)
		break
	}

	// TODO
	cfg.Host = "0.0.0.0"

	buf := new(bytes.Buffer)
	tp := template.Must(template.New("tmplPackageJSON").Parse(tmplPackageJSON))
	if err := tp.Execute(buf, &cfg); err != nil {
		glog.Fatal(err)
	}
	txt := buf.String()

	if err := toFile(txt, *configPath); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", *configPath)
}

type configuration struct {
	NgServeStartCommand     string
	NgServeStartProdCommand string
	Host                    string
}

const tmplPackageJSON = `{
    "name": "app-deephardway",
    "version": "0.9.0",
    "license": "Apache-2.0",
    "angular-cli": {},
    "scripts": {
        "start": "{{.NgServeStartCommand}} --port 4200 --host 0.0.0.0 --proxy-config proxy.config.json",
        "start-prod": "{{.NgServeStartProdCommand}} --port 4200 --host {{.Host}} --proxy-config proxy.config.json",
        "lint": "tslint \"frontend/**/*.ts\"",
        "test": "ng test",
        "pree2e": "webdriver-manager update",
        "e2e": "protractor"
    },
    "private": true,
    "dependencies": {
        "@angular/common": "4.0.3",
        "@angular/compiler": "4.0.3",
        "@angular/compiler-cli": "4.0.3",
        "@angular/core": "4.0.3",
        "@angular/forms": "4.0.3",
        "@angular/http": "4.0.3",
        "@angular/platform-browser": "4.0.3",
        "@angular/platform-browser-dynamic": "4.0.3",
        "@angular/animations": "4.0.3",
        "@angular/router": "4.0.3",
        "@angular/tsc-wrapped": "4.0.3",
        "@angular/upgrade": "4.0.3",
        "core-js": "2.4.1",
        "rxjs": "5.4.0",
        "ts-helpers": "^1.1.1",
        "zone.js": "0.8.11"
    },
    "devDependencies": {
        "@angular/cli": "1.0.6",
        "@types/angular": "^1.5.16",
        "@types/angular-animate": "^1.5.5",
        "@types/angular-cookies": "^1.4.2",
        "@types/angular-mocks": "^1.5.5",
        "@types/angular-resource": "^1.5.6",
        "@types/angular-route": "^1.3.2",
        "@types/angular-sanitize": "^1.3.3",
        "@angular/material": "2.0.0-beta.6",
        "@types/hammerjs": "2.0.34",
        "@types/jasmine": "^2.2.30",
        "@types/node": "^6.0.42",
        "codelyzer": "3.0.1",
        "jasmine-core": "2.4.1",
        "jasmine-spec-reporter": "2.5.0",
        "karma": "1.2.0",
        "karma-chrome-launcher": "^2.0.0",
        "karma-cli": "^1.0.1",
        "karma-jasmine": "^1.0.2",
        "karma-remap-istanbul": "^0.2.1",
        "protractor": "4.0.3",
        "ts-node": "1.2.1",
        "tslint": "5.3.2",
        "typescript": "2.2.0"
    },
    "description": "website",
    "main": "index.js",
    "repository": {
        "url": "git@github.com:gyuho/deephardway.git",
        "type": "git"
    },
    "author": "Gyu-Ho Lee <gyuhox@gmail.com>"
}
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
