package main

import (
	"bytes"
	"flag"
	"strings"
	"text/template"
	"time"

	"github.com/gyuho/dplearn/pkg/fileutil"
	"github.com/gyuho/dplearn/pkg/gcp"

	"github.com/golang/glog"
)

func main() {
	outputPath := flag.String("output", "package.json", "Specify package.json output file path.")
	flag.Parse()

	cfg := configuration{
		NgCommandServeStart:     "ng serve",
		NgCommandServeStartProd: "ng serve --prod",

		// 0.0.0.0 means "all IPv4 addresses on the local machine".
		// If a host has two IP addresses, 192.168.1.1 and 10.1.2.1,
		// and a server running on the host listens on 0.0.0.0,
		// it will be reachable at both of those IPs
		// (Source https://en.wikipedia.org/wiki/0.0.0.0).
		Host:         "0.0.0.0",
		HostPort:     4200,
		HostProd:     "0.0.0.0",
		HostProdPort: 4200,
	}

	bts, err := gcp.GetComputeMetadata("instance/network-interfaces/0/access-configs/0/external-ip", 3, 300*time.Millisecond)
	if err != nil {
		glog.Warning(err)
	} else {
		ip := strings.TrimSpace(string(bts))
		glog.Infof("found public host IP %q", ip)

		// TODO: angular-cli does not work with public IP, so need to use 0.0.0.0
		// https://github.com/angular/angular-cli/issues/2587#issuecomment-252586913
		// https://github.com/webpack/webpack-dev-server/issues/882
	}

	buf := new(bytes.Buffer)
	tp := template.Must(template.New("tmplPackageJSON").Parse(tmplPackageJSON))
	if err := tp.Execute(buf, &cfg); err != nil {
		glog.Fatal(err)
	}
	d := buf.Bytes()

	if err := fileutil.WriteToFile(*outputPath, d); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", *outputPath)
}

type configuration struct {
	NgCommandServeStart     string
	NgCommandServeStartProd string
	Host                    string
	HostPort                int
	HostProd                string
	HostProdPort            int
}

const tmplPackageJSON = `{
    "name": "app-dplearn",
    "version": "0.9.9",
    "license": "Apache-2.0",
    "angular-cli": {},
    "bin": {
        "tslint": "./bin/tslint"
    },
    "scripts": {
        "start": "{{.NgCommandServeStart}} --port {{.HostPort}} --host {{.Host}}",
        "start-prod": "{{.NgCommandServeStartProd}} --port {{.HostProdPort}} --host {{.HostProd}} --disable-host-check",
        "lint": "tslint \"frontend/**/*.ts\"",
        "test": "ng test",
        "pree2e": "webdriver-manager update",
        "e2e": "protractor"
    },
    "private": true,
    "dependencies": {
        "@angular/common": "5.0.0-beta.3",
        "@angular/compiler": "5.0.0-beta.3",
        "@angular/compiler-cli": "5.0.0-beta.3",
        "@angular/core": "5.0.0-beta.3",
        "@angular/forms": "5.0.0-beta.3",
        "@angular/http": "5.0.0-beta.3",
        "@angular/platform-browser": "5.0.0-beta.3",
        "@angular/platform-browser-dynamic": "5.0.0-beta.3",
        "@angular/animations": "5.0.0-beta.3",
        "@angular/router": "5.0.0-beta.3",
        "@angular/tsc-wrapped": "5.0.0-beta.3",
        "@angular/upgrade": "5.0.0-beta.3",
        "@angular/cli": "1.4.0-beta.0",
        "@angular/cdk": "2.0.0-beta.8",
        "@angular/material": "2.0.0-beta.8",
        "@types/angular": "1.6.28",
        "@types/angular-animate": "1.5.8",
        "@types/angular-cookies": "1.4.4",
        "@types/angular-mocks": "1.5.10",
        "@types/angular-resource": "1.5.12",
        "@types/angular-route": "1.3.4",
        "@types/angular-sanitize": "1.3.5",
        "@types/node": "8.0.20",
        "@types/hammerjs": "2.0.34",
        "@types/jasmine": "2.5.53",
        "core-js": "2.5.0",
        "rxjs": "5.4.3",
        "typescript": "2.4.2",
        "ts-node": "3.3.0",
        "ts-helpers": "1.1.2",
        "zone.js": "0.8.16"
    },
    "devDependencies": {
        "codelyzer": "3.1.2",
        "jasmine-core": "2.7.0",
        "jasmine-spec-reporter": "4.2.0",
        "karma": "1.7.0",
        "karma-chrome-launcher": "2.2.0",
        "karma-cli": "1.0.1",
        "karma-jasmine": "1.1.0",
        "karma-remap-istanbul": "0.6.0",
        "protractor": "5.1.2",
        "tslint": "5.6.0"
    },
    "description": "website",
    "main": "index.js",
    "repository": {
        "url": "git@github.com:gyuho/dplearn.git",
        "type": "git"
    },
    "author": "Gyu-Ho Lee <gyuhox@gmail.com>"
}
`
