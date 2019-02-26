// Copyright 2017 The Hugo Authors. All rights reserved.
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

package urls

import (
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/tpl/internal"
)

const name = "urls"

func init() {
	f := func(d *deps.Deps) *internal.TemplateFuncsNamespace {
		ctx := New(d)

		ns := &internal.TemplateFuncsNamespace{
			Name:    name,
			Context: func(args ...interface{}) interface{} { return ctx },
		}

		// Absolute URL functions now just call Relative URL funtions;
		// we do all absolute-URL or path-relative-URL creation in postprocessing.
		// The absolute and relative functions should be deprecated, leaving
		// just neutral functions like "URL" and "Ref"

		//ns.AddMethodMapping(ctx.RelURL,
		//	[]string{"absURL"},
		//	[][2]string{},
		//)
		//ns.AddMethodMapping(ctx.RelLangURL,
		//	[]string{"absLangURL"},
		//	[][2]string{},
		//)
		//ns.AddMethodMapping(ctx.RelRef,
		//	[]string{"ref"},
		//	[][2]string{},
		//)
		// Note: AbsURL is now an alias for RelURL
		ns.AddMethodMapping(ctx.RelURL,
			[]string{"relURL", "absURL"},
			[][2]string{},
		)
		// Note: AbsLangURL is now an alias for relLangURL
		ns.AddMethodMapping(ctx.RelLangURL,
			[]string{"relLangURL", "absLangURL"},
			[][2]string{},
		)
		// Note: Ref is now an alias for RelRef
		ns.AddMethodMapping(ctx.RelRef,
			[]string{"relref", "ref"},
			[][2]string{},
		)
		ns.AddMethodMapping(ctx.URLize,
			[]string{"urlize"},
			[][2]string{},
		)

		ns.AddMethodMapping(ctx.Anchorize,
			[]string{"anchorize"},
			[][2]string{
				{`{{ "This is a title" | anchorize }}`, `this-is-a-title`},
			},
		)

		return ns

	}

	internal.AddTemplateFuncsNamespace(f)
}
