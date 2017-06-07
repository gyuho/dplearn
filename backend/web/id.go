package web

import (
	"crypto/sha512"
	"encoding/base64"
	"net/http"
	"strings"
)

func generateUserID(req *http.Request) string {
	ip := getRealIP(req)
	if ip == "" {
		ip = strings.Split(req.RemoteAddr, ":")[0]
	}
	ip = strings.TrimSpace(strings.Replace(ip, ".", "", -1))
	ua := req.UserAgent()
	return ip + classifyUA(ua) + hashSha512(ip+ua)
}

func getRealIP(req *http.Request) string {
	ts := []string{"X-Forwarded-For", "x-forwarded-for", "X-FORWARDED-FOR"}
	for _, k := range ts {
		if v := req.Header.Get(k); v != "" {
			return v
		}
	}
	return ""
}

func hashSha512(s string) string {
	sum := sha512.Sum512([]byte(s))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func classifyUA(ua string) string {
	var (
		us  = ""
		raw = strings.Replace(strings.ToLower(ua), " ", "", -1)
	)

	// OS
	switch {
	case strings.Contains(raw, "linux"):
		us += "linux"
	case strings.Contains(raw, "macintosh") || strings.Contains(raw, "macos"):
		us += "mac"
	case strings.Contains(raw, "windows"):
		us += "window"
	case strings.Contains(raw, "iphone"):
		us += "iphone"
	case strings.Contains(raw, "android"):
		us += "android"
	case len(raw) > 7:
		us += raw[2:7]
	default:
		us += raw
	}

	// browser
	switch {
	case strings.Contains(raw, "firefox/") && !strings.Contains(raw, "seammonkey/"):
		us += "firefox"
	case strings.Contains(raw, ";msie"):
		us += "ie"
	case strings.Contains(raw, "safari/") && !strings.Contains(raw, "chrome/") && !strings.Contains(raw, "chromium/"):
		us += "safari"
	case strings.Contains(raw, "chrome/") || strings.Contains(raw, "chromium/"):
		us += "chrome"
	case len(raw) > 15:
		us += raw[9:14]
	}

	return us
}
