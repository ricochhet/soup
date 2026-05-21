package http

import (
	"fmt"
	"net/http"
)

type (
	ContentTypeValue string
	HeaderKey        string
)

const (
	ContentTypeZip         ContentTypeValue = "application/zip"
	ContentTypeJSON        ContentTypeValue = "application/json; charset=utf-8"
	ContentTypeHTML        ContentTypeValue = "text/html; charset=utf-8"
	ContentTypeText        ContentTypeValue = "text/plain; charset=utf-8"
	ContentTypeEventStream ContentTypeValue = "text/event-stream"
	ContentTypeBinary      ContentTypeValue = "application/octet-stream"

	HeaderContentType         HeaderKey = "Content-Type"
	HeaderContentDisposition  HeaderKey = "Content-Disposition"
	HeaderWWWAuthenticate     HeaderKey = "WWW-Authenticate"
	HeaderCacheControl        HeaderKey = "Cache-Control"
	HeaderXContentTypeOptions HeaderKey = "X-Content-Type-Options"
	HeaderXFrameOptions       HeaderKey = "X-Frame-Options"
	HeaderConnection          HeaderKey = "Connection"
	HeaderXAccelBuffering     HeaderKey = "X-Accel-Buffering"
)

func Header(w http.ResponseWriter, key HeaderKey, value string) { set(w, string(key), value) }

func NoSniff(
	w http.ResponseWriter,
) {
	set(w, string(HeaderXContentTypeOptions), "nosniff")
}

func DenyFrame(
	w http.ResponseWriter,
) {
	set(w, string(HeaderXFrameOptions), "DENY")
}

func NoContent(
	w http.ResponseWriter,
) {
	w.WriteHeader(http.StatusNoContent)
}

func ContentType(w http.ResponseWriter, ct ContentTypeValue) {
	set(w, string(HeaderContentType), string(ct))
}

func ContentDispositionAttachment(w http.ResponseWriter, filename string) {
	set(w, string(HeaderContentDisposition), fmt.Sprintf(`attachment; filename=%q`, filename))
}

func ContentDispositionInline(w http.ResponseWriter, filename string) {
	set(w, string(HeaderContentDisposition), fmt.Sprintf(`inline; filename=%q`, filename))
}

func BasicAuthChallenge(w http.ResponseWriter, realm string) {
	set(w, string(HeaderWWWAuthenticate), fmt.Sprintf(`Basic realm=%q`, realm))
}

func NoCache(w http.ResponseWriter) {
	set(w, string(HeaderCacheControl), "no-store, no-cache, must-revalidate")
}

func SSEHeaders(w http.ResponseWriter) {
	ContentType(w, ContentTypeEventStream)
	Header(w, HeaderCacheControl, "no-cache")
	Header(w, HeaderConnection, "keep-alive")
	Header(w, HeaderXAccelBuffering, "no")
}

func set(w http.ResponseWriter, k, v string) { w.Header().Set(k, v) }
