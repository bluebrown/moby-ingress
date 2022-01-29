package main

import "strings"

func TemplateName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
