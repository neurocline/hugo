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

    "github.com/gohugoio/hugo/bufferpool"
    "github.com/gohugoio/hugo/transform"

    jww "github.com/spf13/jwalterweatherman"
)

// ConvertSiteToPathAbs turns site-absolute-URL links into path-absolute-URL-links
// (markup processors don't know about basePath, the path component of baseURL)
// Fixup for pages moving in the hierarchy (e.g. "pretty urls") is done at
// a higher level.
func ConvertSiteToPathAbs(content []byte, basePath string) []byte {
    if basePath == "" {
        return content
    }

    source := bytes.NewReader(content)

    transformers := transform.NewEmpty()
    transformers = append(transformers, NewAbsURLTransformer(basePath))
    work := bufferpool.GetBuffer()
    defer bufferpool.PutBuffer(work)

    if err := transformers.Apply(work, source); err != nil {
        jww.ERROR.Printf("AdjustMarkupLinks: error transforming buffer: %s\n", err)
        // We can't return an error, so we just return the unmodified content
        return content
    }

    // We have transformed content, copy it from the to-be-recycled buffer.
    // Just doing work.Bytes() isn't safe, because work will be recycled on scope exit.
    // Reuse the existing slice if possible.
    // TODO not a Go guru yet, so this might not be optimal, but it feels like it.
    content = content[:0]
    content = append(content, work.Bytes()...)

    return content
}

