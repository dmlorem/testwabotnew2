package util

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
	"mvdan.cc/xurls/v2"
)

var wapatern = regexp.MustCompile(`(?i)(?:https?:\/\/)?chat\.whatsapp\.com\/+\w{22}`)

func MatchURL(s string) bool {
	rx := xurls.Relaxed()
	idxEmail := rx.SubexpIndex("relaxedEmail")
	matches := rx.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if match[idxEmail] != "" {
			continue
		}
		return true
	}
	return false
}

func MatchWaUrl(s string) bool {
	return wapatern.MatchString(s)
}

func NormalizeString(s string) string {
	t := norm.NFD.String(s)
	var b strings.Builder
	for _, r := range t {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
