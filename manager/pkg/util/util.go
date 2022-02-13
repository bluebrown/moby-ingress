package util

import "strings"

func MockControllerLabels() map[string]string {
	return map[string]string{
		"ingress.class":            "haproxy",
		"ingress.global":           "spread-checks 15\n",
		"ingress.defaults":         "timeout connect 5s\ntimeout check 5s\ntimeout client 2m\ntimeout server 2m\nretries 1\nretry-on all-retryable-errors\noption redispatch 1\ndefault-server check inter 30s\n",
		"ingress.frontend.default": "bind *:3000\n",
	}
}

func TemplateName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
