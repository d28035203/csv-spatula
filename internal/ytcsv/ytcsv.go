// Package ytcsv holds shared helpers for YouTube Takeout CSV tools.
package ytcsv

import (
	"errors"
	"io"
	"strings"
	"time"
	"unicode"
)

// IsVideoID reports whether s looks like a classic YouTube video id (11 chars).
func IsVideoID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 11 {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// WatchURL builds a youtube.com watch URL for id.
func WatchURL(id string) string {
	return "https://www.youtube.com/watch?v=" + id
}

// ParseTimeUnix parses common Takeout / RFC3339 timestamps to unix seconds.
func ParseTimeUnix(isoStr string) int64 {
	isoStr = strings.TrimSpace(isoStr)
	if isoStr == "" {
		return 0
	}
	if t, err := time.Parse(time.RFC3339, isoStr); err == nil {
		return t.Unix()
	}
	// Common Takeout variant: +00:00 already valid RFC3339; try without subseconds noise
	isoStr = strings.ReplaceAll(isoStr, "+00:00", "Z")
	if t, err := time.Parse(time.RFC3339, isoStr); err == nil {
		return t.Unix()
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, isoStr); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// EscapeHTML escapes text for use inside HTML attributes/text nodes.
func EscapeHTML(s string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&#39;",
	)
	return replacer.Replace(s)
}

// IsEOF reports whether err is end-of-file from a reader.
func IsEOF(err error) bool {
	return errors.Is(err, io.EOF)
}
