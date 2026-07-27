package utils

import (
	"net/http"
	"strings"
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

// TestMapsContainOnlyInertTypes is the standing guard on this package's single
// security property: an extension may steer the answer, but only ever into a
// type that cannot execute script or render markup in the app's origin.
//
// The two maps are checked together because they are the only two paths by
// which the uploader-supplied file name influences the result, and because the
// hazard is a future edit rather than today's contents — someone adding a
// convenient ".xml" or ".svg" entry would silently turn CasaDrop into a
// stored-XSS vector, and nothing else in the suite would notice.
func TestMapsContainOnlyInertTypes(t *testing.T) {
	forbidden := []string{"text/", "image/svg", "application/xml", "application/xhtml", "+xml", "javascript", "application/pdf"}

	check := func(mapName, ext, mime string) {
		lower := strings.ToLower(mime)
		for _, bad := range forbidden {
			if strings.Contains(lower, bad) {
				t.Errorf("%s[%q] = %q — not an inert media type; this map may only promote into audio/* or video/*", mapName, ext, mime)
			}
		}
		if !strings.HasPrefix(lower, "audio/") && !strings.HasPrefix(lower, "video/") {
			t.Errorf("%s[%q] = %q — expected an audio/ or video/ type", mapName, ext, mime)
		}
	}
	for ext, mime := range safeMediaByExt {
		check("safeMediaByExt", ext, mime)
	}
	for ext, mime := range ambiguousByExt {
		check("ambiguousByExt", ext, mime)
	}

	// Every extension listed as ambiguous must have a resolution, otherwise the
	// lookup silently falls through and keeps the wrong sniffed type. (There is
	// deliberately no mime.TypeByExtension fallback — see the map's comment.)
	for sniffed, exts := range ambiguousSniffs {
		for ext := range exts {
			if ambiguousByExt[ext] == "" {
				t.Errorf("ambiguousSniffs[%q] lists %q but ambiguousByExt has no entry — the override silently does nothing", sniffed, ext)
			}
		}
	}
}

// TestRefineMimeType_AmbiguousContainers covers the cases where a *recognised*
// sniff is overridden, which is the one place the usual "sniffing wins" rule
// does not apply.
func TestRefineMimeType_AmbiguousContainers(t *testing.T) {
	tests := []struct {
		sniffed, name, want, why string
	}{
		{"video/webm", "Movie.mkv", "video/x-matroska", "EBML magic is identical for MKV and WebM"},
		{"video/webm", "Track.mka", "audio/x-matroska", "Matroska audio"},
		{"video/webm", "Voice.weba", "audio/webm", "WebM audio was rendered as <video> before"},
		{"video/webm", "Clip.webm", "video/webm", "a genuine WebM must stay untouched"},
		{"video/mp4", "Song.m4a", "audio/mp4", "M4A carries an ftyp box and sniffs as video/mp4"},
		{"video/mp4", "Book.m4b", "audio/mp4", "audiobooks likewise"},
		{"video/mp4", "Clip.mp4", "video/mp4", "a real MP4 must stay video"},
		{"video/mp4", "Clip.m4v", "video/mp4", "m4v is video and is not in the override list"},
		// The override must not become a general-purpose lever.
		{"video/mp4", "evil.html", "video/mp4", "an unlisted extension cannot steer a recognised type"},
		{"text/html; charset=utf-8", "song.m4a", "text/html; charset=utf-8", "html is not ambiguous with anything"},
		{"image/svg+xml", "song.mka", "image/svg+xml", "active types are never reachable"},
	}
	for _, tt := range tests {
		if got := RefineMimeType(tt.sniffed, tt.name); got != tt.want {
			t.Errorf("RefineMimeType(%q, %q) = %q, want %q — %s", tt.sniffed, tt.name, got, tt.want, tt.why)
		}
	}
}

func TestHasBrowserPlaybackCaveat(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"video/x-matroska", true},
		{"audio/x-matroska", true},
		{"VIDEO/X-MATROSKA", true},              // case-insensitive
		{"video/x-matroska; codecs=hevc", true}, // parameters stripped
		{"video/mp4", false},
		{"audio/flac", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := HasBrowserPlaybackCaveat(tt.in); got != tt.want {
			t.Errorf("HasBrowserPlaybackCaveat(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// TestSourceTypeAttr_WaveAlias pins a measured browser behaviour: Chrome's
// canPlayType("audio/wave") returns "" (a flat reject) while "audio/wav"
// returns "maybe". Go's sniffer produces the former spelling, so emitting it
// verbatim would make the browser discard the source without ever fetching it —
// the exact failure the attribute is supposed to prevent.
func TestSourceTypeAttr_WaveAlias(t *testing.T) {
	if got := SourceTypeAttr("audio/wave"); got != "audio/wav" {
		t.Errorf("SourceTypeAttr(audio/wave) = %q, want audio/wav", got)
	}
	if got := SourceTypeAttr("AUDIO/WAVE"); got != "audio/wav" {
		t.Errorf("SourceTypeAttr(AUDIO/WAVE) = %q, want audio/wav", got)
	}
}

func TestServingMimeType(t *testing.T) {
	// An unset stored type must never reach the wire as an empty Content-Type,
	// or the browser sniffs its own way to a conclusion we did not sanction.
	for _, in := range []string{"", "   "} {
		if got := ServingMimeType(in); got != "application/octet-stream" {
			t.Errorf("ServingMimeType(%q) = %q, want application/octet-stream", in, got)
		}
	}
	if got := ServingMimeType("audio/flac"); got != "audio/flac" {
		t.Errorf("ServingMimeType(audio/flac) = %q, want it unchanged", got)
	}
}
