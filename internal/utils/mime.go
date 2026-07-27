package utils

import (
	"net/http"
	"path/filepath"
	"strings"
)

// ambiguousSniffs maps a sniffed MIME type to the file extensions that
// http.DetectContentType provably cannot distinguish from it.
//
// Matroska: .mkv, .mka and .weba share the same EBML magic bytes (1A 45 DF A3)
// with .webm, so the sniffer reports "video/webm" for all of them. A 4K HEVC
// movie in an MKV then gets labelled as WebM — a container that only permits
// VP8/VP9/AV1 with Vorbis/Opus. Lenient browsers play it anyway, strict ones
// refuse outright, and downloads carry a wrong Content-Type. A .weba (WebM
// audio) got the same treatment in reverse: labelled video/webm, so the share
// page rendered a <video> element for a file that has no picture.
//
// ISO-BMFF: .m4a and .m4b carry an `ftyp` box just like .mp4, so the sniffer
// commits to "video/mp4" for all of them (measured for both the iTunes
// `ftypM4A …M4A mp42isom` and the ffmpeg `ftypisom…iso2mp41` layouts). An audio
// book or music file then renders as a black <video> rectangle. This is the one
// place where an extension overrides a *recognised* type — see the safety note
// on ambiguousByExt for why that is still sound.
var ambiguousSniffs = map[string]map[string]bool{
	"video/webm": {".mkv": true, ".mk3d": true, ".mka": true, ".mks": true, ".weba": true},
	"video/mp4":  {".m4a": true, ".m4b": true},
}

// ambiguousByExt resolves the extensions listed above to their real type.
//
// SAFETY: every value here must be an inert media type, exactly like
// safeMediaByExt — this map is the only thing that can override a type the
// sniffer already committed to, so an active type (text/html, image/svg+xml,
// application/xml, …) must never appear in it. There is deliberately no
// mime.TypeByExtension fallback: it consults the OS mime database, which would
// happily answer "image/svg+xml" for an extension somebody adds to
// ambiguousSniffs without adding it here, quietly turning this map's safety
// property into a hope. An extension with no entry here keeps the sniffed type.
var ambiguousByExt = map[string]string{
	".mkv":  "video/x-matroska",
	".mk3d": "video/x-matroska",
	".mka":  "audio/x-matroska",
	".mks":  "video/x-matroska",
	".weba": "audio/webm",
	".m4a":  "audio/mp4",
	".m4b":  "audio/mp4",
}

// nonCommittalSniffs are the answers http.DetectContentType gives when it
// recognises nothing — "I don't know", not "it is this". Go only carries
// signatures for a handful of media formats, so FLAC, M4A and an MP3 without an
// ID3 header all land here, and application/ogg is too coarse to tell audio
// from video. Without help these files get MediaTypeUnknown and the share page
// renders no player at all.
var nonCommittalSniffs = map[string]bool{
	"application/octet-stream": true,
	"application/ogg":          true,
}

// safeMediaByExt is the ONLY promotion allowed out of a non-committal sniff.
// Every value is an inert media type: none of them can execute script or render
// markup in the app's origin, so trusting the uploader-supplied extension this
// far cannot turn into content-type confusion. Types that can (text/html,
// image/svg+xml, application/xml, …) are deliberately absent and must never be
// added — for those the sniffer is authoritative.
// Two entries deserve a note, because both look removable and are not:
//
//   - Extensions whose *genuine* files sniff to a committal type (.wav →
//     audio/wave, .avi → video/avi, .webm/.weba → video/webm, most .mp4 →
//     video/mp4) never reach this map for a well-formed upload. They are kept
//     as the fallback for files the sniffer cannot place — a truncated or
//     otherwise unrecognised MP4 does land on application/octet-stream
//     (measured), and without the entry it would be a download with no player.
//   - `.ogg` maps to audio deliberately, although the container can also hold
//     Theora video. Ogg video conventionally uses .ogv (which is listed
//     separately), and the alternative is worse than the mistake it avoids:
//     leaving application/ogg in place means getMediaType sees an
//     `application/` prefix and the share page renders no player at all. An
//     audio player for a rare Theora file is a smaller failure than no player
//     for the common Vorbis one.
var safeMediaByExt = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".m4b":  "audio/mp4",
	".aac":  "audio/aac",
	".oga":  "audio/ogg",
	".opus": "audio/ogg",
	".ogg":  "audio/ogg",
	".wav":  "audio/wav",
	".weba": "audio/webm",
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".webm": "video/webm",
	".ogv":  "video/ogg",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".mkv":  "video/x-matroska",
	".mka":  "audio/x-matroska",
}

// UndetectedMimeType is what the writers store when detection never ran at all
// — the file could not be opened or could not be read. It is deliberately NOT
// "application/octet-stream": that string is also a genuine sniff result ("I
// looked and recognised nothing"), and conflating the two made the stored-data
// migration promote rows on the strength of the uploader-supplied file name
// alone, for files whose bytes nobody had ever looked at. Those rows then went
// from download-only to rendered inline. An empty value is excluded from the
// migration's candidate set for free, because a SQL `IN` list matches neither
// the empty string nor NULL.
const UndetectedMimeType = ""

// ServingMimeType turns a stored type into one that is safe to put on the wire.
// An unset type becomes application/octet-stream, so a response never carries an
// empty Content-Type for the browser to sniff its own way out of.
func ServingMimeType(stored string) string {
	if strings.TrimSpace(stored) == "" {
		return "application/octet-stream"
	}
	return stored
}

// DetectMimeType returns the MIME type for the given content, using the file
// name only to resolve ambiguities the content sniffer is known to have.
//
// Sniffing stays authoritative on purpose: the file name comes from the
// uploader, and deriving the type from it wholesale would invite content-type
// confusion (an "evil.html" sniffed as text/plain but served as text/html).
// The extension is consulted *only* when the sniffed type is listed in
// ambiguousSniffs and the extension is one of the alternatives that sniff
// identically — so it can never promote a file into an unrelated type.
func DetectMimeType(content []byte, fileName string) string {
	return RefineMimeType(http.DetectContentType(content), fileName)
}

// RefineMimeType applies the same disambiguation to an *already sniffed* type.
//
// It exists so the stored-data migration can correct rows written by older
// versions without re-reading every uploaded file: the value in the database is
// exactly what http.DetectContentType returned back then, so feeding it back in
// yields the same answer DetectMimeType would produce today. The safety
// properties are identical — a row that sniffed as text/html is not
// non-committal and can never be promoted.
func RefineMimeType(sniffed, fileName string) string {
	base := sniffed
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}

	ext := strings.ToLower(filepath.Ext(fileName))

	// Case 1: the sniffer recognised nothing. Promote only into the inert-media
	// allow-list; anything else keeps the sniffed type and stays a download.
	if nonCommittalSniffs[base] {
		if t := safeMediaByExt[ext]; t != "" {
			return t
		}
		return sniffed
	}

	// Case 2: the sniffer answered, but provably cannot distinguish this type
	// from the listed alternatives (Matroska vs. WebM, M4A vs. MP4).
	alts, ok := ambiguousSniffs[base]
	if !ok || !alts[ext] {
		return sniffed
	}
	if t := ambiguousByExt[ext]; t != "" {
		return t
	}
	return sniffed
}

// browserPlayableTypes are the media types a browser can meaningfully evaluate
// via canPlayType(). Handing <source type="…"> anything else is worse than
// omitting the attribute: the browser rejects the source without ever fetching
// it, so a file it could actually have played never gets a chance.
var browserPlayableTypes = map[string]bool{
	"video/mp4":   true,
	"video/webm":  true,
	"video/ogg":   true,
	"audio/mpeg":  true,
	"audio/mp4":   true,
	"audio/aac":   true,
	"audio/ogg":   true,
	"audio/wav":   true,
	"audio/x-wav": true,
	"audio/webm":  true,
	"audio/flac":  true,
}

// caveatTypes are containers a browser may render only partially — typically
// because the container commonly carries codecs no browser licenses. Matroska
// is the practical case: it routinely holds AC-3/E-AC-3/DTS audio, for which
// Chrome, Firefox and Edge ship no decoder, so the video plays silently.
//
// This is deliberately about the *container*, not the codec: the server does
// not demux uploads, so it cannot know the actual audio codec. Saying "playback
// may be limited here, download for the full file" is honest; claiming to know
// the codec would not be.
var caveatTypes = map[string]bool{
	"video/x-matroska": true,
	"audio/x-matroska": true,
}

// HasBrowserPlaybackCaveat reports whether the share page should warn that
// in-browser playback may be incomplete for this type.
func HasBrowserPlaybackCaveat(mimeType string) bool {
	base := mimeType
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	return caveatTypes[strings.ToLower(base)]
}

// attrAliases rewrite a stored type into the spelling browsers actually accept.
//
// Go's sniffer returns "audio/wave" for a RIFF/WAVE file, but measured in
// Chrome, `canPlayType("audio/wave")` returns the empty string — a flat reject —
// while "audio/wav" returns "maybe". Emitting the sniffed spelling verbatim
// would therefore make the browser discard the source without ever fetching it,
// which is precisely the failure the comment on browserPlayableTypes warns
// about. Only the attribute is rewritten; the stored type stays as sniffed.
var attrAliases = map[string]string{
	"audio/wave": "audio/wav",
}

// SourceTypeAttr returns the value for a <source type="…"> attribute, or ""
// when the type should be omitted so the browser sniffs the stream itself.
func SourceTypeAttr(mimeType string) string {
	base := mimeType
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	base = strings.ToLower(base)
	if alias, ok := attrAliases[base]; ok {
		return alias
	}
	if browserPlayableTypes[base] {
		return mimeType
	}
	return ""
}
