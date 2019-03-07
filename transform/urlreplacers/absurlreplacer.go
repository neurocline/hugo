// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package urlreplacers

import (
	"bytes"
	"io"
	"path/filepath"
	"unicode"
	"unicode/utf8"

	"github.com/gohugoio/hugo/transform"

//	jww "github.com/spf13/jwalterweatherman"
)

type absurllexer struct {
	// the source to absurlify
	content []byte
	// the target for the new absurlified content
	w io.Writer

	// path may be set to a "." relative path
	path []byte

	kind URLKind
	targetPath []byte // path to the file being scanned
	baseURL []byte
	basePath []byte

	pos   int // input position
	start int // item start position

	quotes [][]byte
}

type stateFunc func(*absurllexer) stateFunc

type prefix struct {
	disabled bool
	b        []byte
	f        func(l *absurllexer)
}

func newPrefixState() []*prefix {
	return []*prefix{
		{b: []byte("src="), f: checkCandidateBase},
		{b: []byte("href="), f: checkCandidateBase},
		{b: []byte("srcset="), f: checkCandidateSrcset},
	}
}

//type absURLMatcher struct {
//	match []byte
//	quote []byte
//}

func (l *absurllexer) emit() {
	l.w.Write(l.content[l.start:l.pos])
	l.start = l.pos
}

var (
	relURLPrefix    = []byte("/")
	relURLPrefixLen = len(relURLPrefix)
)

func (l *absurllexer) consumeQuote() []byte {
	for _, q := range l.quotes {
		if bytes.HasPrefix(l.content[l.pos:], q) {
			l.pos += len(q)
			l.emit()
			return q
		}
	}
	return nil
}

// handle URLs in src and href.
// TBD - handle non-path-absolute URLs in links
func checkCandidateBase(l *absurllexer) {
	l.consumeQuote()
	quotedStart := l.pos // start of path inside quotes
	quoteChar := l.content[l.pos-1:l.pos]

	if !bytes.HasPrefix(l.content[l.pos:], relURLPrefix) {
		return
	}

	// check for schemaless URLs
	posAfter := l.pos + relURLPrefixLen
	if posAfter >= len(l.content) {
		return
	}
	r, _ := utf8.DecodeRune(l.content[posAfter:])
	if r == '/' {
		// schemaless: skip
		return
	}
	if l.pos > l.start {
		l.emit()
	}

	// If we are really handling relative URLs properly, then we have to get
	// the link, so we can adjust it relative to this file.
	if l.kind == PathRelativeURL {
//		jww.WARN.Printf("Convert to PathRelativeURL\n")

		// TBD handle malformed text, this will eat up the whole file
		for l.pos < len(l.content) && l.content[l.pos] != quoteChar[0] {
			l.pos++
		}
		if l.pos == len(l.content) {
			return
		}

		link := make([]byte, l.pos - quotedStart, 32 + l.pos - quotedStart) // add space for long relative path
		copy(link, l.content[quotedStart:l.pos])
		l.pos += len(quoteChar)
//		jww.WARN.Printf("  link=%s\n", string(link))

		// If we have a path-absolute link, then make it relative.
		// Ignore other links for now.
		if link[0] == '/' {
			// Get a copy that we can munge
			target := make([]byte, len(l.targetPath))
			copy(target, l.targetPath)
//			jww.WARN.Printf("  target=%s\n", string(target))

			// Remove the common prefix of link and target
			n := len(link)
			if n > len(target) {
				n = len(target)
			}
			i := 0
			for ; i < n; i++ {
				if link[i] != target[i] {
					break
				}
			}
			link = link[i:]
			target = target[i:]
//			jww.WARN.Printf("  suffix link=%s\n", string(link))
//			jww.WARN.Printf("  suffix target=%s\n", string(target))

			// For each directory left in target, prepend "../" to link
			for {
				pos := bytes.LastIndexByte(target, '/')
				if pos == -1 {
					break
				}
				link = append(link, 0, 0, 0)
				copy(link[3:], link[0:])
				copy(link[0:3], []byte("../"))
				//link = append([]byte("../"), link...)
				target = target[:pos]
			}
//			jww.WARN.Printf("  relative link=%s\n", string(link))

			// Now write out the new link, properly quoted
			l.w.Write(link)
			l.w.Write(quoteChar)
			l.start = l.pos
		}

		// Emit any pending bytes (only for unhandled cases, and we probably don't
		// need to do this, it's done in the outer loop too).
		if l.pos > l.start {
			l.emit()
		}

		return
	}

	// Old method
	l.pos += relURLPrefixLen
	l.w.Write(l.path)
	l.start = l.pos
}

func (l *absurllexer) posAfterURL(q []byte) int {
	if len(q) > 0 {
		// look for end quote
		return bytes.Index(l.content[l.pos:], q)
	}

	return bytes.IndexFunc(l.content[l.pos:], func(r rune) bool {
		return r == '>' || unicode.IsSpace(r)
	})

}

// handle URLs in srcset.
func checkCandidateSrcset(l *absurllexer) {
	q := l.consumeQuote()
	if q == nil {
		// srcset needs to be quoted.
		return
	}

	// special case, not frequent (me think)
	if !bytes.HasPrefix(l.content[l.pos:], relURLPrefix) {
		return
	}

	// check for schemaless URLs
	posAfter := l.pos + relURLPrefixLen
	if posAfter >= len(l.content) {
		return
	}
	r, _ := utf8.DecodeRune(l.content[posAfter:])
	if r == '/' {
		// schemaless: skip
		return
	}

	posEnd := l.posAfterURL(q)

	// safe guard
	if posEnd < 0 || posEnd > 2000 {
		return
	}

	if l.pos > l.start {
		l.emit()
	}

	section := l.content[l.pos : l.pos+posEnd+1]

	fields := bytes.Fields(section)
	for i, f := range fields {
		if f[0] == '/' {
			l.w.Write(l.path)
			l.w.Write(f[1:])

		} else {
			l.w.Write(f)
		}

		if i < len(fields)-1 {
			l.w.Write([]byte(" "))
		}
	}

	l.pos += len(section)
	l.start = l.pos

}

// main loop
func (l *absurllexer) replace() {
	contentLength := len(l.content)

	prefixes := newPrefixState()

	for {
		if l.pos >= contentLength {
			break
		}

		nextPos := -1

		var match *prefix

		for _, p := range prefixes {
			if p.disabled {
				continue
			}
			idx := bytes.Index(l.content[l.pos:], p.b)

			if idx == -1 {
				p.disabled = true
				// Find the closest match
			} else if nextPos == -1 || idx < nextPos {
				nextPos = idx
				match = p
			}
		}

		if nextPos == -1 {
			// Done!
			l.pos = contentLength
			break
		} else {
			l.pos += nextPos + len(match.b)
			match.f(l)
		}
	}

	// Done!
	if l.pos > l.start {
		l.emit()
	}
}

func doReplace(path string, ct transform.FromTo, quotes [][]byte) {

	lexer := &absurllexer{
		content: ct.From().Bytes(),
		w:       ct.To(),
		path:    []byte(path),
		quotes:  quotes}

	lexer.replace()
}

func doReplace2(kind URLKind, targetPath string, baseURL string, basePath string, ct transform.FromTo, quotes [][]byte) {

	// Add basePath to targetPath (the caller gave us an output-relative path)
	if basePath != "" {
		targetPath = filepath.Join(basePath, targetPath)
	}
	targetPath = filepath.ToSlash(targetPath)

	lexer := &absurllexer{
		content: ct.From().Bytes(),
		w:       ct.To(),
		kind:    kind,
		targetPath: []byte(targetPath),
		baseURL: []byte(baseURL),
		basePath: []byte(basePath),
		quotes:  quotes}

	lexer.replace()
}

type absURLReplacer struct {
	htmlQuotes [][]byte
	xmlQuotes  [][]byte
}

func newAbsURLReplacer() *absURLReplacer {
	return &absURLReplacer{
		htmlQuotes: [][]byte{[]byte("\""), []byte("'")},
		xmlQuotes:  [][]byte{[]byte("&#34;"), []byte("&#39;")}}
}

func (au *absURLReplacer) replaceInHTML(path string, ct transform.FromTo) {
	doReplace(path, ct, au.htmlQuotes)
}

func (au *absURLReplacer) replaceInXML(path string, ct transform.FromTo) {
	doReplace(path, ct, au.xmlQuotes)
}

type urlReplacer struct {
	htmlQuotes [][]byte
	xmlQuotes  [][]byte
}

func newURLReplacer() *urlReplacer {
	return &urlReplacer{
		htmlQuotes: [][]byte{[]byte("\""), []byte("'")},
		xmlQuotes:  [][]byte{[]byte("&#34;"), []byte("&#39;")}}
}

func (au *urlReplacer) replaceInHTML(kind URLKind, targetPath string, baseURL string, basePath string, ct transform.FromTo) {
	doReplace2(kind, targetPath, baseURL, basePath, ct, au.htmlQuotes)
}
