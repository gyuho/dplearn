package web

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
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
	return fmt.Sprintf("%s_%s_%s", ip, classifyUA(ua), hashSha512(ip + ua)[:30])
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

func classifyUA(ua string) (s string) {
	raw := strings.Replace(strings.ToLower(ua), " ", "", -1)

	// OS
	switch {
	case strings.Contains(raw, "linux"):
		s += "linux"
	case strings.Contains(raw, "macintosh") || strings.Contains(raw, "macos"):
		s += "mac"
	case strings.Contains(raw, "windows"):
		s += "window"
	case strings.Contains(raw, "iphone"):
		s += "iphone"
	case strings.Contains(raw, "android"):
		s += "android"
	case len(raw) > 7:
		s += raw[2:7]
	default:
		s += raw
	}

	// browser
	switch {
	case strings.Contains(raw, "firefox/") && !strings.Contains(raw, "seammonkey/"):
		s += "firefox"
	case strings.Contains(raw, ";msie"):
		s += "ie"
	case strings.Contains(raw, "safari/") && !strings.Contains(raw, "chrome/") && !strings.Contains(raw, "chromium/"):
		s += "safari"
	case strings.Contains(raw, "chrome/") || strings.Contains(raw, "chromium/"):
		s += "chrome"
	case len(raw) > 15:
		s += raw[9:14]
	}

	return s
}
