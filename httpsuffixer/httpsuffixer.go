// Copyright 2019 The Reserve Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpsuffixer

import (
	"net/http"
	"strings"
)

type SuffixServer struct {
	NewTweaker func(contentType string) Tweaker
}

type Tweaker interface {
	Tweak(data []byte) []byte
}

type responseWriter struct {
	Server *SuffixServer
	Parent http.ResponseWriter

	tweaker Tweaker
}

func (w *responseWriter) Header() http.Header {
	return w.Parent.Header()
}

func (w *responseWriter) Write(data []byte) (int, error) {
	oLen := len(data)
	if w.tweaker != nil {
		data = w.tweaker.Tweak(data)
	}
	bytesWritten, err := w.Parent.Write(data)
	if err != nil {
		return bytesWritten, err
	}
	return oLen, nil
}

func (w *responseWriter) WriteHeader(statusCode int) {
	contentType := strings.SplitN(w.Parent.Header().Get("Content-Type"), ";", 2)[0]
	if tweaker := w.Server.NewTweaker(contentType); tweaker != nil {
		w.tweaker = tweaker
		w.Header().Del("Content-Length") // TODO
	}
	w.Parent.WriteHeader(statusCode)
}

func (w *responseWriter) Flush() {
	w.Parent.(http.Flusher).Flush()
}

func (w *responseWriter) Finish() {
	if w.tweaker == nil {
		return
	}
	if trailer := w.tweaker.Tweak(nil); trailer != nil {
		w.Parent.Write(trailer)
	}
}

func (s *SuffixServer) WrapServer(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrappedWriter := responseWriter{Server: s, Parent: w}
		handler.ServeHTTP(&wrappedWriter, r)
		wrappedWriter.Finish()
	})
}
