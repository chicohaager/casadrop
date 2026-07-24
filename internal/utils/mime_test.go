package utils

import (
	"net/http"
	"testing"
)

// ebml is the Matroska/WebM magic. Both containers start with it, which is why
// http.DetectContentType cannot tell them apart.
var ebml = []byte{0x1A, 0x45, 0xDF, 0xA3, 0x93, 0x42, 0x82, 0x88}

func TestDetectMimeType_MatroskaIsNotWebM(t *testing.T) {
	// Guard the premise: if Go ever learns to distinguish the two, this
	// disambiguation becomes unnecessary and the test should tell us.
	if got := http.DetectContentType(ebml); got != "video/webm" {
		t.Fatalf("premise changed: http.DetectContentType(EBML) = %q, want video/webm", got)
	}

	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{"mkv is matroska", "Movie.mkv", "video/x-matroska"},
		{"mkv uppercase ext", "MOVIE.MKV", "video/x-matroska"},
		{"mka is matroska audio", "Album.mka", "audio/x-matroska"},
		{"webm stays webm", "clip.webm", "video/webm"},
		{"unrelated ext keeps sniffed type", "clip.mp4", "video/webm"},
		{"no extension keeps sniffed type", "clip", "video/webm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectMimeType(ebml, tt.fileName); got != tt.want {
				t.Errorf("DetectMimeType(%q) = %q, want %q", tt.fileName, got, tt.want)
			}
		})
	}
}

// The extension must never be able to promote a file into an unrelated type —
// that is the content-type-confusion hole we are deliberately not opening.
func TestDetectMimeType_ExtensionCannotOverrideUnrelatedContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		fileName string
		want     string
	}{
		{"html named mkv stays html", []byte("<!DOCTYPE html><html><body>x</body></html>"), "evil.mkv", "text/html; charset=utf-8"},
		{"png named mkv stays png", []byte("\x89PNG\r\n\x1a\n"), "evil.mkv", "image/png"},
		{"script named mkv stays text", []byte("#!/bin/sh\necho hi\n"), "evil.mkv", "text/plain; charset=utf-8"},
		{"pdf named mkv stays pdf", []byte("%PDF-1.7\n"), "evil.mkv", "application/pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectMimeType(tt.content, tt.fileName); got != tt.want {
				t.Errorf("DetectMimeType(%q) = %q, want %q", tt.fileName, got, tt.want)
			}
		})
	}
}

// Go carries signatures for only a few media formats. FLAC, M4A and an MP3
// without an ID3 header all sniff as application/octet-stream, which made the
// share page render no player at all.
func TestDetectMimeType_PromotesUnrecognisedMedia(t *testing.T) {
	octet := []byte("\x00\x01\x02\x03binary payload the sniffer does not know")
	tests := []struct {
		fileName string
		want     string
	}{
		{"song.flac", "audio/flac"},
		{"song.mp3", "audio/mpeg"},
		{"song.m4a", "audio/mp4"},
		{"clip.mov", "video/quicktime"},
		{"clip.avi", "video/x-msvideo"},
		{"movie.mkv", "video/x-matroska"},
		// Not media → stays a download.
		{"archive.zip", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			if got := DetectMimeType(octet, tt.fileName); got != tt.want {
				t.Errorf("DetectMimeType(%q) = %q, want %q", tt.fileName, got, tt.want)
			}
		})
	}
}

// The promotion must never reach a type that can execute script or render
// markup in the app's origin — that is the whole reason for the allow-list.
func TestDetectMimeType_PromotionCannotReachActiveTypes(t *testing.T) {
	octet := []byte("\x00\x01\x02\x03binary payload the sniffer does not know")
	for _, name := range []string{"evil.html", "evil.htm", "evil.svg", "evil.xml", "evil.js", "evil.xhtml"} {
		if got := DetectMimeType(octet, name); got != "application/octet-stream" {
			t.Errorf("DetectMimeType(%q) = %q, want application/octet-stream", name, got)
		}
	}
	// And a real HTML payload keeps its sniffed type regardless of extension.
	html := []byte("<!DOCTYPE html><html><body>x</body></html>")
	if got := DetectMimeType(html, "song.flac"); got != "text/html; charset=utf-8" {
		t.Errorf("html named .flac = %q, want text/html", got)
	}
}

func TestSourceTypeAttr(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"video/mp4", "video/mp4"},
		{"video/webm", "video/webm"},
		{"audio/mpeg", "audio/mpeg"},
		// Browsers return "" from canPlayType() for these, and a <source> whose
		// type they reject is never fetched — so the attribute must be omitted.
		{"video/x-matroska", ""},
		{"video/vnd.avi", ""},
		{"video/quicktime", ""},
		{"application/octet-stream", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := SourceTypeAttr(tt.in); got != tt.want {
			t.Errorf("SourceTypeAttr(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
