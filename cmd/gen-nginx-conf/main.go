package main

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gyuho/deephardway/pkg/fileutil"
	"github.com/gyuho/deephardway/pkg/gcp"

	"github.com/golang/glog"
)

func main() {
	outputPath := flag.String("output", "nginx.conf", "Specify nginx.conf output file path.")
	targetPort := flag.Int("target-port", 4200, "Specify target host port to proxy requests to.")
	flag.Parse()

	cfg := configuration{
		ServerName: "deephardway.com",
		TargetPort: *targetPort,
	}

	bts, err := gcp.GetComputeMetadata("instance/network-interfaces/0/access-configs/0/external-ip", 3, 300*time.Millisecond)
	if err != nil {
		glog.Warning(err)
	} else {
		ip := strings.TrimSpace(string(bts))
		glog.Infof("found public host IP %q", ip)
		cfg.ServerName = ip + " " + cfg.ServerName
	}

	buf := new(bytes.Buffer)
	tp := template.Must(template.New("tmplNginxConf").Parse(tmplNginxConf))
	if err := tp.Execute(buf, &cfg); err != nil {
		glog.Fatal(err)
	}
	d := buf.Bytes()

	if err := fileutil.WriteToFile(*outputPath, d); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", *outputPath)

	glog.Infof("writing to /etc/nginx/sites-available/default")
	if err = os.MkdirAll("/etc/nginx/sites-available/", os.ModePerm); err != nil {
		glog.Fatal(err)
	}
	if err = fileutil.WriteToFile("/etc/nginx/sites-available/default", d); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote to /etc/nginx/sites-available/default")
	/*
	   // Configure reverse proxy
	   RUN mkdir -p /etc/nginx/sites-available/
	   ADD nginx.conf /etc/nginx/sites-available/default
	*/
}

type configuration struct {
	ServerName string
	TargetPort int
}

const tmplNginxConf = `server {
	listen 80;

	access_log /var/log/nginx/access.log;
	error_log /var/log/nginx/error.log;

	set_real_ip_from 0.0.0.0/0;
	real_ip_header X-Forwarded-For;
	real_ip_recursive on;
	server_name {{.ServerName}};

	location / {
		proxy_read_timeout 3000s;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $remote_addr;
		proxy_pass http://127.0.0.1:{{.TargetPort}};
	}
}
`
