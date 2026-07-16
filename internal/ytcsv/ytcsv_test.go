package ytcsv

import (
	"io"
	"testing"
)

func TestIsVideoID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"dQw4w9WgXcQ", true},
		{"abcdefghijk", true},
		{"short", false},
		{"toolongvideoidx", false},
		{"bad id!!!!!", false},
		{"  dQw4w9WgXcQ  ", true},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsVideoID(tc.in); got != tc.want {
			t.Fatalf("IsVideoID(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseTimeUnix(t *testing.T) {
	if ParseTimeUnix("") != 0 {
		t.Fatal("empty should be 0")
	}
	// 2020-01-01T00:00:00Z
	got := ParseTimeUnix("2020-01-01T00:00:00Z")
	if got != 1577836800 {
		t.Fatalf("got %d", got)
	}
}

func TestEscapeHTML(t *testing.T) {
	got := EscapeHTML(`A & B <C> "x"`)
	want := "A &amp; B &lt;C&gt; &quot;x&quot;"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsEOF(t *testing.T) {
	if !IsEOF(io.EOF) {
		t.Fatal("expected EOF")
	}
}
