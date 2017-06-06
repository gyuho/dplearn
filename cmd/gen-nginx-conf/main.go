package main

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gyuho/deephardway/pkg/gcp"

	"github.com/golang/glog"
)

func main() {
	outputPath := flag.String("output", "nginx.conf", "Specify nginx.conf output file path.")
	flag.Parse()

	cfg := configuration{
		ServerName: "deephardway.com",
		HostPort:   4200,
	}

	for i := 0; i < 3; i++ {
		// inspect metadata to get public IP
		bts, err := gcp.GetComputeMetadata("instance/network-interfaces/0/access-configs/0/external-ip")
		if err != nil {
			glog.Warning(err)
			time.Sleep(300 * time.Millisecond)
			continue
		}
		ip := strings.TrimSpace(string(bts))
		glog.Infof("found public host IP %q", ip)
		cfg.ServerName = ip + " " + cfg.ServerName
		break
	}

	buf := new(bytes.Buffer)
	tp := template.Must(template.New("tmplNginxConf").Parse(tmplNginxConf))
	if err := tp.Execute(buf, &cfg); err != nil {
		glog.Fatal(err)
	}
	txt := buf.String()

	if err := toFile(txt, *outputPath); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("wrote %q", *outputPath)
}

type configuration struct {
	ServerName string
	HostPort   int
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
		proxy_pass http://127.0.0.1:{{.HostPort}};
	}
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
