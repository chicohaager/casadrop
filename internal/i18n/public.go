// Package i18n holds the strings of the server-rendered pages: what a recipient
// or guest sees (share, folder, receive, not found), the login and setup pages,
// and the receive-upload error messages. Those are rendered server-side, so
// the language comes from the visitor's browser (Accept-Language), not from the
// admin UI's localStorage setting. English is the fallback for every key.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Fallback is used when no supported language matches.
const Fallback = "en"

// public holds one string table per language, loaded from locales/<lang>.json.
// The set of languages must equal the admin UI's SUPPORTED_LANGS (app.js) —
// TestLanguagesMatchAdminUI enforces it, so a language can't be half-added.
var public = load()

//go:embed locales/*.json
var localeFS embed.FS

func load() map[string]map[string]string {
	entries, err := localeFS.ReadDir("locales")
	if err != nil {
		panic("i18n: " + err.Error())
	}
	out := make(map[string]map[string]string, len(entries))
	for _, e := range entries {
		b, err := localeFS.ReadFile("locales/" + e.Name())
		if err != nil {
			panic("i18n: " + err.Error())
		}
		table := map[string]string{}
		if err := json.Unmarshal(b, &table); err != nil {
			panic("i18n: " + e.Name() + ": " + err.Error())
		}
		out[strings.TrimSuffix(e.Name(), ".json")] = table
	}
	return out
}

// Supported reports whether lang has its own string table.
func Supported(lang string) bool {
	_, ok := public[lang]
	return ok
}

// Strings returns the complete string table for lang: every key of the
// fallback language, overridden by lang's own entries where they exist.
func Strings(lang string) map[string]string {
	out := make(map[string]string, len(public[Fallback]))
	for k, v := range public[Fallback] {
		out[k] = v
	}
	for k, v := range public[lang] {
		out[k] = v
	}
	return out
}

// T returns the string for key in lang, English if lang lacks it, the key
// itself if nobody has it (visible, so a typo is noticed, never blank).
func T(lang, key string) string {
	if v, ok := public[lang][key]; ok {
		return v
	}
	if v, ok := public[Fallback][key]; ok {
		return v
	}
	return key
}

// Format returns T(lang, key) with every {name} replaced by vars[name].
// Whole sentences with named slots keep word order and grammar right in
// every language (no "Max " + n + " uploads" concatenation).
func Format(lang, key string, vars map[string]string) string {
	pairs := make([]string, 0, 2*len(vars))
	for k, v := range vars {
		pairs = append(pairs, "{"+k+"}", v)
	}
	// One pass: a substituted value is never scanned again, so a file name
	// containing "{date}" stays literal whatever the map order.
	return strings.NewReplacer(pairs...).Replace(T(lang, key))
}

// FormatTime renders t with the language's layout (key format.dateTime).
func FormatTime(lang string, t time.Time) string {
	return t.Format(T(lang, "format.dateTime"))
}

// lookup returns table[key], or the key itself when it is missing — a typo
// in a template then shows up as a visible key instead of an empty string.
func lookup(table map[string]string, key string) string {
	if v, ok := table[key]; ok {
		return v
	}
	return key
}

// TemplateFuncs are the helpers the server-rendered pages use:
//
//	{{tf .T "folder.files" "n" .TotalFiles}}           → plain string (auto-escaped)
//	{{tfh .T "receive.uploadsUsed" "*cur" 3 "max" 20}} → HTML; a "*" name is bolded
//	{{fdate .Lang .ExpiresAtTime}}                     → localized date/time
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"tf": func(table map[string]string, key string, kv ...interface{}) string {
			pairs := []string{}
			for i := 0; i+1 < len(kv); i += 2 {
				pairs = append(pairs, "{"+strings.TrimPrefix(fmt.Sprint(kv[i]), "*")+"}", fmt.Sprint(kv[i+1]))
			}
			return strings.NewReplacer(pairs...).Replace(lookup(table, key))
		},
		"tfh": func(table map[string]string, key string, kv ...interface{}) template.HTML {
			pairs := []string{}
			for i := 0; i+1 < len(kv); i += 2 {
				name := fmt.Sprint(kv[i])
				val := template.HTMLEscapeString(fmt.Sprint(kv[i+1]))
				if strings.HasPrefix(name, "*") {
					name, val = name[1:], "<strong>"+val+"</strong>"
				}
				pairs = append(pairs, "{"+name+"}", val)
			}
			return template.HTML(strings.NewReplacer(pairs...).Replace(template.HTMLEscapeString(lookup(table, key))))
		},
		"fdate": func(lang string, t time.Time) string { return FormatTime(lang, t) },
	}
}

// Lang picks the language for a public page: an explicit, supported ?lang=
// query parameter wins; otherwise the best supported Accept-Language entry by
// q-value; otherwise the fallback.
func Lang(r *http.Request) string {
	if q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang"))); Supported(q) {
		return q
	}
	return fromAcceptLanguage(r.Header.Get("Accept-Language"))
}

func fromAcceptLanguage(header string) string {
	type cand struct {
		lang string
		q    float64
		pos  int
	}
	var cs []cand
	for i, part := range strings.Split(header, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		tag := strings.ToLower(strings.TrimSpace(fields[0]))
		if tag == "" || tag == "*" {
			continue
		}
		q := 1.0
		for _, f := range fields[1:] {
			f = strings.TrimSpace(f)
			if strings.HasPrefix(f, "q=") {
				if v, err := strconv.ParseFloat(f[2:], 64); err == nil {
					q = v
				}
			}
		}
		if q <= 0 {
			continue
		}
		base := tag
		if i := strings.IndexAny(tag, "-_"); i > 0 {
			base = tag[:i]
		}
		cs = append(cs, cand{base, q, i})
	}
	sort.SliceStable(cs, func(a, b int) bool { return cs[a].q > cs[b].q })
	for _, c := range cs {
		if Supported(c.lang) {
			return c.lang
		}
	}
	return Fallback
}
