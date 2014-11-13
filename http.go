package main

import (
	"regexp"
	"strings"
)

var httpMethods []string = []string{"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "POST", "PUT", "TRACE"}
var httpMethodStr string = strings.Join(httpMethods, "|")
var httpre *regexp.Regexp = regexp.MustCompile("(?i)(" + httpMethodStr + ") ")

func IsHttp(c Conn) bool {
	p, err := c.Peek(20)
	if err != nil {
		return false
	}
	return httpre.Match(p)
}
