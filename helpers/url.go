// Copyright 2015 The Hugo Authors. All rights reserved.
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

package helpers

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/purell"
	jww "github.com/spf13/jwalterweatherman"
)

type pathBridge struct {
}

func (pathBridge) Base(in string) string {
	return path.Base(in)
}

func (pathBridge) Clean(in string) string {
	return path.Clean(in)
}

func (pathBridge) Dir(in string) string {
	return path.Dir(in)
}

func (pathBridge) Ext(in string) string {
	return path.Ext(in)
}

func (pathBridge) Join(elem ...string) string {
	return path.Join(elem...)
}

func (pathBridge) Separator() string {
	return "/"
}

var pb pathBridge

func sanitizeURLWithFlags(in string, f purell.NormalizationFlags) string {
	s, err := purell.NormalizeURLString(in, f)
	if err != nil {
		return in
	}

	// Temporary workaround for the bug fix and resulting
	// behavioral change in purell.NormalizeURLString():
	// a leading '/' was inadvertently added to relative links,
	// but no longer, see #878.
	//
	// I think the real solution is to allow Hugo to
	// make relative URL with relative path,
	// e.g. "../../post/hello-again/", as wished by users
	// in issues #157, #622, etc., without forcing
	// relative URLs to begin with '/'.
	// Once the fixes are in, let's remove this kludge
	// and restore SanitizeURL() to the way it was.
	//                         -- @anthonyfok, 2015-02-16
	//
	// Begin temporary kludge
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	if len(u.Path) > 0 && !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}
	return u.String()
	// End temporary kludge

	//return s

}

// SanitizeURL sanitizes the input URL string.
func SanitizeURL(in string) string {
	return sanitizeURLWithFlags(in, purell.FlagsSafe|purell.FlagRemoveTrailingSlash|purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes|purell.FlagRemoveUnnecessaryHostDots|purell.FlagRemoveEmptyPortSeparator)
}

// SanitizeURLKeepTrailingSlash is the same as SanitizeURL, but will keep any trailing slash.
func SanitizeURLKeepTrailingSlash(in string) string {
	return sanitizeURLWithFlags(in, purell.FlagsSafe|purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes|purell.FlagRemoveUnnecessaryHostDots|purell.FlagRemoveEmptyPortSeparator)
}

// URLize is similar to MakePath, but with Unicode handling
// Example:
//     uri: Vim (text editor)
//     urlize: vim-text-editor
func (p *PathSpec) URLize(uri string) string {
	return p.URLEscape(p.MakePathSanitized(uri))

}

// URLizeFilename creates an URL from a filename by esacaping unicode letters
// and turn any filepath separator into forward slashes.
func (p *PathSpec) URLizeFilename(filename string) string {
	return p.URLEscape(filepath.ToSlash(filename))
}

// URLEscape escapes unicode letters.
func (p *PathSpec) URLEscape(uri string) string {
	// escape unicode letters
	parsedURI, err := url.Parse(uri)
	if err != nil {
		// if net/url can not parse URL it means Sanitize works incorrectly
		panic(err)
	}
	x := parsedURI.String()
	return x
}

// MakePermalink combines base URL with content path to create full URL paths.
// Example
//    base:   http://spf13.com/
//    path:   post/how-i-blog
//    result: http://spf13.com/post/how-i-blog
func MakePermalink(host, plink string) *url.URL {

	base, err := url.Parse(host)
	if err != nil {
		panic(err)
	}

	p, err := url.Parse(plink)
	if err != nil {
		panic(err)
	}

	if p.Host != "" {
		panic(fmt.Errorf("Can't make permalink from absolute link %q", plink))
	}

	base.Path = path.Join(base.Path, p.Path)

	// path.Join will strip off the last /, so put it back if it was there.
	hadTrailingSlash := (plink == "" && strings.HasSuffix(host, "/")) || strings.HasSuffix(p.Path, "/")
	if hadTrailingSlash && !strings.HasSuffix(base.Path, "/") {
		base.Path = base.Path + "/"
	}

	return base
}

// AbsURL creates an absolute URL from the relative path given and the BaseURL set in config.
func (p *PathSpec) AbsURL(in string, addLanguage bool) string {
	url, err := url.Parse(in)
	if err != nil {
		return in
	}

	if url.IsAbs() || strings.HasPrefix(in, "//") {
		return in
	}

	var baseURL string
	if strings.HasPrefix(in, "/") {
		u := p.BaseURL.URL()
		u.Path = ""
		baseURL = u.String()
	} else {
		baseURL = p.BaseURL.String()
	}

	if addLanguage {
		prefix := p.GetLanguagePrefix()
		if prefix != "" {
			hasPrefix := false
			// avoid adding language prefix if already present
			if strings.HasPrefix(in, "/") {
				hasPrefix = strings.HasPrefix(in[1:], prefix)
			} else {
				hasPrefix = strings.HasPrefix(in, prefix)
			}

			if !hasPrefix {
				addSlash := in == "" || strings.HasSuffix(in, "/")
				in = path.Join(prefix, in)

				if addSlash {
					in += "/"
				}
			}
		}
	}
	return MakePermalink(baseURL, in).String()
}

// IsAbsURL determines whether the given path points to an absolute URL.
func IsAbsURL(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	return url.IsAbs() || strings.HasPrefix(path, "//")
}

// RelURL creates a path-absolute-URL or an absolute-URL and turns it
// into a path-absolute-URL with an optional language inserted between
// the basePath and the rest of the link. The output path is as faithful
// to the input path as possible.
//
// TODO - tpl/urls.RelUrl seems to think this will turn a path-relative
// URL into a path-absolute-URL, but there's no way that can happen, as
// we don't have access to the file that hosts this link. And that should
// be done in tpl/urls.RelUrl anyway.
// TODO - this code was written to satisfy the spec implied by the tests.
// But that spec is probably wrong. Get agreement on this and then revisit.
func (p *PathSpec) RelURL(in string, addLanguage bool) string {

	// If this is a schemaless URL, or it's an http absolute-URL not pointing inside
	// our side, there's nothing to do.
	// TODO - this will break if we are handed URLs with other schemes
	baseURL := p.BaseURL.String()
	if (!strings.HasPrefix(in, baseURL) && strings.HasPrefix(in, "http")) || strings.HasPrefix(in, "//") {
		return in
	}

	// Get a sanitized basePath - split off any trailing slash, because
	// that will go after any language dir
	basePath := p.GetBasePath()
	var trailingSlash string
	if strings.HasSuffix(basePath, "/") {
		trailingSlash = "/"
		basePath = basePath[:len(basePath)-1]
	}

	// Get the language prefix (if we are adding a langauge). We have one of
	// no language: langPrefix=""
	// language: langPrefix = "/" + lang
	// This simplifies creating the path
	var langPrefix string
	if addLanguage {
		langPrefix = p.GetLanguagePrefix()
		if langPrefix != "" {
			langPrefix = "/" + langPrefix
		}
	}

	// If we were handed an empty input path, then just return a basepath
	// (with any trailing slash after the language dir).
	// TODO - this should probably be an error
	if in == "" {
		return basePath + langPrefix + trailingSlash
	}

	// If we were handed an absolute-URL, then turn it into a site-absolute-URL
	// (by stripping host+BasePath)
	link := in
	if strings.HasPrefix(link, baseURL) {
		link = trailingSlash + strings.TrimPrefix(link, baseURL)
	}

	// If we were handed a site-relative path, force it to be site-absolute
	// (and note that this is a huge mistake, we are going to end up with
	// a path that only works in one specific case, that of the user actually
	// passing in a site-absolute path but omitting the leading slash - e.g.
	// bad user input)
	if !strings.HasPrefix(link, "/") {
		link = "/" + link
	}

	// If the input path has the language prefix, don't add it a second time
	if langPrefix != "" && strings.HasPrefix(link[1:], langPrefix[1:]) {
		langPrefix = ""
	}

	// Assemble it all together to make a path-absolute-URL with any
	// desired language path following right after the BasePath portion
	link = basePath + langPrefix + link

	return link
}

// PrependBasePath prepends baseURL.BasePath to the given path
// (input path is presumed to be a site-absolute-URL)
func (p *PathSpec) PrependBasePath(rel string) string {
	basePath := p.GetBasePath()
	if basePath != "" {
		// TODO- remove this when all Hugo paths are normalized to slash form.
		rel = filepath.ToSlash(rel)
		// path.Join will remove any trailing slash, so we need to keep a note.
		hadSlash := strings.HasSuffix(rel, "/")
		rel = path.Join(basePath, rel)
		if hadSlash {
			rel += "/"
		}
	}
	if !strings.HasPrefix(rel, "/") && !strings.HasPrefix(rel, "\\") {
		jww.ERROR.Printf("PrependBasePath expects site-absolute-URL: %s\n", rel)
	}
	return rel
}

// URLizeAndPrep applies misc sanitation to the given URL to get it in line
// with the Hugo standard.
func (p *PathSpec) URLizeAndPrep(in string) string {
	return p.URLPrep(p.URLize(in))
}

// URLPrep applies misc sanitation to the given URL.
func (p *PathSpec) URLPrep(in string) string {
	if p.UglyURLs {
		return Uglify(SanitizeURL(in))
	}
	pretty := PrettifyURL(SanitizeURL(in))
	if path.Ext(pretty) == ".xml" {
		return pretty
	}
	url, err := purell.NormalizeURLString(pretty, purell.FlagAddTrailingSlash)
	if err != nil {
		return pretty
	}
	return url
}

// PrettifyURL takes a URL string and returns a semantic, clean URL.
func PrettifyURL(in string) string {
	x := PrettifyURLPath(in)

	if path.Base(x) == "index.html" {
		return path.Dir(x)
	}

	if in == "" {
		return "/"
	}

	return x
}

// PrettifyURLPath takes a URL path to a content and converts it
// to enable pretty URLs.
//     /section/name.html       becomes /section/name/index.html
//     /section/name/           becomes /section/name/index.html
//     /section/name/index.html becomes /section/name/index.html
func PrettifyURLPath(in string) string {
	return prettifyPath(in, pb)
}

// Uglify does the opposite of PrettifyURLPath().
//     /section/name/index.html becomes /section/name.html
//     /section/name/           becomes /section/name.html
//     /section/name.html       becomes /section/name.html
func Uglify(in string) string {
	if path.Ext(in) == "" {
		if len(in) < 2 {
			return "/"
		}
		// /section/name/  -> /section/name.html
		return path.Clean(in) + ".html"
	}

	name, ext := fileAndExt(in, pb)
	if name == "index" {
		// /section/name/index.html -> /section/name.html
		d := path.Dir(in)
		if len(d) > 1 {
			return d + ext
		}
		return in
	}
	// /.xml -> /index.xml
	if name == "" {
		return path.Dir(in) + "index" + ext
	}
	// /section/name.html -> /section/name.html
	return path.Clean(in)
}
