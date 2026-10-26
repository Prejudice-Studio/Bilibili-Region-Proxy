// Package proxy implements the bilibili.com regional HTTP resolver.
package proxy

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────
// Requests handled here are for the Bilibili main site.
// ─────────────────────────────────────────────────────────────────────────
const (
	apiBase  = "https://api.bilibili.com"
	passport = "https://passport.bilibili.com"
)

type route struct {
	needle  string
	base    string
	referer bool
}

// Fixed endpoints: client path -> upstream URL (paths differ between the two).
var fixedRoutes = []route{
	{"/auth/set_cookie_qrcode", passport + "/x/passport-login/web/qrcode/poll", true},
}

// Host-level routes: keep the client path, only swap the host.
var hostRoutes = []route{
	{"/x/passport-login/", passport, true},
}

var hopByHop = map[string]bool{
	"connection": true, "keep-alive": true, "proxy-authenticate": true,
	"proxy-authorization": true, "te": true, "trailer": true,
	"transfer-encoding": true, "upgrade": true,
}

var dropRequest = func() map[string]bool {
	out := map[string]bool{
		"host": true, "content-length": true,
		"x-forwarded-for": true, "x-forwarded-host": true, "x-forwarded-proto": true,
		"resolver_mode": true, "x-resolver-mode": true, "x-resolver_mode": true,
	}
	for k := range hopByHop {
		out[k] = true
	}
	return out
}()

var (
	seaSearchPaths = map[string]bool{
		"/x/web-interface/wbi/search/type": true,
		"/x/web-interface/search/type":     true,
		"/x/v2/search/type":                true,
	}
	normalSearchModes = map[string]bool{"CN": true, "HK": true, "MO": true, "TW": true}
	seaResolverModes  = map[string]bool{
		"INTL": true, "SEA": true,
		"TH": true, "SG": true, "MY": true, "ID": true,
		"PH": true, "VN": true, "MM": true, "KH": true,
		"LA": true, "BN": true, "TL": true,
		"THAILAND": true, "SINGAPORE": true, "MALAYSIA": true,
		"INDONESIA": true, "PHILIPPINES": true, "VIETNAM": true,
		"SOUTHEAST_ASIA": true, "SOUTHEAST-ASIA": true,
	}

	resolverQueryParams = map[string]bool{
		"resolver_mode":   true,
		"resolver_cursor": true,
		"x-resolver-mode": true,
		"x-resolver_mode": true,
	}
)

func useIntlBangumiSearch(mode string) bool {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode == "" {
		return false
	}
	if normalSearchModes[mode] {
		return false
	}
	if seaResolverModes[mode] {
		return true
	}
	// Keep custom resolver modes working as before; only CN/HK/MO/TW are
	// treated as ordinary searches.
	return true
}

// ─────────────────────────────────────────────────────────────────────────
// HTTP client for upstream requests
// ─────────────────────────────────────────────────────────────────────────

var upstreamClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		DisableCompression:    true,
		TLSClientConfig:       &tls.Config{},
	},
}

// ─────────────────────────────────────────────────────────────────────────
// HTTP helpers
// ─────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}

func errorJSON(statusCode int, code int, message string) map[string]any {
	return map[string]any{"code": code, "message": message}
}

func pickUpstream(path, query string) (string, bool) {
	qs := ""
	if query != "" {
		qs = "?" + query
	}
	for _, r := range fixedRoutes {
		if strings.Contains(path, r.needle) {
			return r.base + qs, r.referer
		}
	}
	for _, r := range hostRoutes {
		if strings.Contains(path, r.needle) {
			return r.base + path + qs, r.referer
		}
	}
	// Fallback: forward the path as-is to bilibili.com so no request silently
	// escapes the resolver (unknown paths get the upstream's own 404).
	return apiBase + path + qs, true
}

func cleanRequestHeaders(h http.Header) http.Header {
	out := http.Header{}
	for k, vals := range h {
		if dropRequest[strings.ToLower(k)] {
			continue
		}
		for _, v := range vals {
			out.Add(k, v)
		}
	}
	return out
}

// cleanResponseHeaders drops hop-by-hop headers plus compression metadata:
// the body is forwarded already decoded, so content-encoding/content-length
// must not survive.
func cleanResponseHeaders(h http.Header) http.Header {
	out := http.Header{}
	for k, vals := range h {
		lk := strings.ToLower(k)
		if hopByHop[lk] || lk == "content-encoding" || lk == "content-length" {
			continue
		}
		for _, v := range vals {
			out.Add(k, v)
		}
	}
	return out
}

func stripResolverQueryParams(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	kept := make([]string, 0, 8)
	for _, component := range strings.Split(rawQuery, "&") {
		if component == "" {
			continue
		}
		name := component
		if idx := strings.IndexByte(component, '='); idx >= 0 {
			name = component[:idx]
		}
		decoded, err := url.QueryUnescape(name)
		if err != nil {
			decoded = name
		}
		if resolverQueryParams[decoded] {
			continue
		}
		kept = append(kept, component)
	}
	return strings.Join(kept, "&")
}

func resolverModeFromRequest(r *http.Request) (string, bool) {
	for _, name := range []string{"resolver_mode", "x-resolver-mode", "x-resolver_mode"} {
		if header := strings.TrimSpace(r.Header.Get(name)); header != "" {
			return header, true
		}
	}
	if value, ok := r.URL.Query()["resolver_mode"]; ok && len(value) > 0 {
		return value[0], true
	}
	return "", false
}

// ─────────────────────────────────────────────────────────────────────────
// Main proxy handler
// ─────────────────────────────────────────────────────────────────────────

// Handler proxies ordinary HTTP traffic and routes SEA bangumi searches through
// the localized Bilibili search page.
func Handler(w http.ResponseWriter, r *http.Request) {
	requestPath := r.URL.Path

	// Explicit SEA resolver URL form. The server's own egress determines the
	// region; this prefix only labels the client's selected resolver.
	if strings.HasPrefix(requestPath, "/resolver/") {
		parts := strings.SplitN(requestPath, "/", 4)
		if len(parts) != 4 || parts[2] != "sea" || parts[3] == "" {
			writeJSON(w, http.StatusNotFound, errorJSON(-404, -404, "Unknown resolver route."))
			return
		}
		requestPath = "/" + parts[3]
	}

	requestQuery := stripResolverQueryParams(r.URL.RawQuery)
	forceIntlBangumiSearch := loadProxyConfig().forceIntlBangumiSearch
	resolverMode, hasResolverMode := resolverModeFromRequest(r)
	if forceIntlBangumiSearch || hasResolverMode {
		// resolver_mode remains a local routing hint and never goes upstream.
		useIntlSearch := forceIntlBangumiSearch || useIntlBangumiSearch(resolverMode)
		if useIntlSearch && seaSearchPaths[requestPath] && r.URL.Query().Get("search_type") == "media_bangumi" {
			proxyIntlBangumiSearch(w, r)
			return
		}
	}

	requestHeaders := cleanRequestHeaders(r.Header)
	requestHeaders.Set("Accept-Encoding", "identity")

	upstreamURL, needReferer := pickUpstream(requestPath, requestQuery)
	if needReferer {
		requestHeaders.Set("Referer", "https://www.bilibili.com/")
	}

	// APP playback: BBDown marks platform=android; the APP UA is required.
	if strings.Contains(requestPath, "playurl") && strings.Contains(requestQuery, "platform=android") {
		requestHeaders.Set("User-Agent", "Bilibili Freedoooooom/MarkII")
	} else if seaSearchPaths[requestPath] {
		// The search endpoint requires buvid3; BBDown puts it in the query.
		if buvid3 := r.URL.Query().Get("buvid3"); buvid3 != "" {
			requestHeaders.Set("Cookie", "buvid3="+buvid3)
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, "Upstream did not respond."))
		return
	}
	// Never log the query string: playurl URLs may carry short-lived signatures
	// or account/device identifiers.
	log.Printf("%s %s -> %s", r.Method, requestPath, strings.SplitN(upstreamURL, "?", 2)[0])

	upstreamRequest, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, "Upstream did not respond."))
		return
	}
	upstreamRequest.Header = requestHeaders

	upstream, err := upstreamClient.Do(upstreamRequest)
	if err != nil {
		log.Printf("Upstream request failed (%T)", err)
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, "Upstream did not respond."))
		return
	}
	defer upstream.Body.Close()

	// Apply configured additions only to successful ordinary search JSON. The
	// original numResults count remains the upstream count.
	upstreamContentType := strings.ToLower(upstream.Header.Get("Content-Type"))
	if seaSearchPaths[requestPath] &&
		isFirstSearchPage(r.URL.Query()) &&
		upstream.StatusCode >= 200 && upstream.StatusCode < 300 &&
		strings.Contains(upstreamContentType, "json") {
		additions, _ := configuredSearchInjections(nil)
		if len(additions) > 0 {
			payloadBytes, readErr := io.ReadAll(upstream.Body)
			var payload map[string]any
			if readErr == nil && json.Unmarshal(payloadBytes, &payload) == nil && payload != nil {
				payload = injectSearchResults(payload, nil)
				writeResponseHeaders(w, cleanResponseHeaders(upstream.Header))
				w.WriteHeader(upstream.StatusCode)
				_ = json.NewEncoder(w).Encode(payload)
				return
			}
			log.Println("Search response was not valid JSON; injection was skipped")
			writeResponseHeaders(w, cleanResponseHeaders(upstream.Header))
			w.WriteHeader(upstream.StatusCode)
			_, _ = w.Write(payloadBytes)
			return
		}
	}

	writeResponseHeaders(w, cleanResponseHeaders(upstream.Header))
	w.WriteHeader(upstream.StatusCode)

	bodyReader := decodeUpstreamBody(upstream)
	flushWriter := &flushingWriter{w: w}
	if flusher, ok := w.(http.Flusher); ok {
		flushWriter.flusher = flusher
	}
	_, _ = io.Copy(flushWriter, bodyReader)
}

func writeResponseHeaders(w http.ResponseWriter, headers http.Header) {
	dst := w.Header()
	for k, vals := range headers {
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

// decodeUpstreamBody transparently decodes gzip/deflate bodies so the stripped
// content-encoding header never disagrees with the bytes on the wire.
func decodeUpstreamBody(upstream *http.Response) io.Reader {
	switch strings.ToLower(strings.TrimSpace(upstream.Header.Get("Content-Encoding"))) {
	case "gzip":
		if zr, err := gzip.NewReader(upstream.Body); err == nil {
			return zr
		}
	case "deflate":
		if zr, err := zlib.NewReader(upstream.Body); err == nil {
			return zr
		}
	}
	return upstream.Body
}

type flushingWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (fw *flushingWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return n, err
}
