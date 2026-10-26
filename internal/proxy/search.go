package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const (
	intlSearchPageURL   = "https://search.bilibili.com/zh-Hans/bangumi"
	maxSearchPageBytes  = 8 << 20
	intlSearchTimeout   = 20 * time.Second
	intlSearchEvalLimit = 2 * time.Second
)

func intlSearchQuery(input url.Values, keyword string, page int) url.Values {
	query := make(url.Values, len(input)+3)
	for name, values := range input {
		query[name] = append([]string(nil), values...)
	}
	for name := range resolverQueryParams {
		query.Del(name)
	}
	query.Set("keyword", keyword)
	query.Set("category", "bangumi")
	query.Set("page", strconv.Itoa(page))
	return query
}

func intlSearchHeaders(input http.Header) http.Header {
	headers := cleanRequestHeaders(input)
	if headers.Get("Accept") == "" {
		headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	}
	if headers.Get("Accept-Language") == "" {
		headers.Set("Accept-Language", "zh-Hans-SG,zh-Hans;q=0.9,zh;q=0.8,en;q=0.5")
	}
	if headers.Get("User-Agent") == "" {
		headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
	}
	if headers.Get("Referer") == "" {
		headers.Set("Referer", "https://search.bilibili.com/")
	}
	// The response is parsed as plain bytes. Avoid brotli/gzip compression
	// supplied by a client header that the parser cannot decode here.
	headers.Set("Accept-Encoding", "identity")
	return headers
}

var intlSearchPageClient = &http.Client{
	Transport:     upstreamClient.Transport,
	Timeout:       intlSearchTimeout,
	CheckRedirect: preserveSearchQueryOnRedirect,
}

// preserveSearchQueryOnRedirect keeps the search parameters when Bilibili
// canonicalizes /zh-Hans/bangumi to /bangumi. Some edge nodes return a
// relative Location without a query string; following it as-is turns the
// request into an empty search page.
func preserveSearchQueryOnRedirect(next *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return fmt.Errorf("too many search page redirects")
	}
	if len(via) == 0 || next.URL == nil {
		return nil
	}
	previous := via[len(via)-1]
	if next.URL.RawQuery == "" && previous.URL != nil && previous.URL.RawQuery != "" {
		next.URL.RawQuery = previous.URL.RawQuery
		next.URL.ForceQuery = previous.URL.ForceQuery
	}
	// Keep the same browser-like headers on the canonical request. Go copies
	// most headers automatically, but explicitly restoring these avoids a
	// locale change on redirect implementations that rebuild the request.
	for _, name := range []string{"Accept", "Accept-Language", "User-Agent", "Referer"} {
		if value := previous.Header.Get(name); value != "" {
			if next.Header == nil {
				next.Header = make(http.Header)
			}
			next.Header.Set(name, value)
		}
	}
	return nil
}

func proxyIntlBangumiSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorJSON(-405, -405, "Use GET for the ordinary search request."))
		return
	}
	query := r.URL.Query()
	keyword := strings.TrimSpace(query.Get("keyword"))
	if keyword == "" || len([]rune(keyword)) > 100 {
		writeJSON(w, http.StatusBadRequest, errorJSON(-400, -400, "Invalid search keyword."))
		return
	}
	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(query.Get("page_size"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	pageURL, _ := url.Parse(intlSearchPageURL)
	// Keep the client's complete query set so account/device/search options
	// survive the web-search adaptation. Resolver fields are local routing
	// controls and must never be sent upstream.
	pageQuery := intlSearchQuery(query, keyword, page)
	pageURL.RawQuery = pageQuery.Encode()

	upstreamRequest, err := http.NewRequestWithContext(r.Context(), http.MethodGet, pageURL.String(), nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, "Search page request could not be created."))
		return
	}
	// Forward the client's headers, including Cookie, Authorization, access-key
	// related headers and device metadata. cleanRequestHeaders removes only
	// hop-by-hop, forwarding and resolver-control headers. Defaults are used
	// only when the client did not provide a browser header.
	upstreamRequest.Header = intlSearchHeaders(r.Header)

	upstream, err := intlSearchPageClient.Do(upstreamRequest)
	if err != nil {
		logSearchError(err)
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, "Search page did not respond."))
		return
	}
	defer upstream.Body.Close()
	body, err := io.ReadAll(io.LimitReader(upstream.Body, maxSearchPageBytes+1))
	if err != nil || len(body) > maxSearchPageBytes {
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, "Search page response was too large or incomplete."))
		return
	}
	if upstream.StatusCode < 200 || upstream.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, fmt.Sprintf("Search page returned HTTP %d.", upstream.StatusCode)))
		return
	}

	response, region, err := intlBangumiResponse(body, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorJSON(-502, -502, "Search page data could not be parsed."))
		return
	}
	data := mapValue(response["data"])
	upstreamCount := len(asSlice(data["result"]))
	if page == 1 {
		response = injectSearchResults(response, nil)
	}
	returnedCount := len(asSlice(mapValue(response["data"])["result"]))
	log.Printf("Intl web bangumi search completed region=%s page=%d upstream_results=%d injected_results=%d", region, page, upstreamCount, returnedCount-upstreamCount)
	writeJSON(w, http.StatusOK, response)
}

func logSearchError(err error) {
	// Avoid writing request URLs, which contain the search keyword.
	log.Printf("Intl web search request failed (%T)", err)
}

func intlBangumiResponse(document []byte, requestedPage, pageSize int) (map[string]any, string, error) {
	root, err := intlSearchState(document)
	if err != nil {
		return nil, "", err
	}
	searchResponse := mapValue(root["searchResponse"])
	searchType := mapValue(mapValue(root["searchTypeResponse"])["searchTypeResponse"])
	searchAll := mapValue(searchResponse["searchAllResponse"])
	results := make([]any, 0)
	seen := make(map[string]bool)
	appendSearchTypeResults(&results, seen, searchType, true)
	if len(results) == 0 {
		appendSearchTypeResults(&results, seen, searchAll, false)
	}

	total := searchTypeTotal(searchType)
	if total == 0 {
		total = mediaBangumiTotal(searchAll)
	}
	if total < int64(len(results)) {
		total = int64(len(results))
	}
	pages := 0
	if total > 0 {
		pages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	response := map[string]any{
		"code":    0,
		"message": "0",
		"ttl":     1,
		"data": map[string]any{
			"numResults": total,
			"page":       requestedPage,
			"pages":      pages,
			"numPages":   pages,
			"page_size":  pageSize,
			"result":     results,
		},
	}
	return response, searchPageRegion(root), nil
}

func intlSearchState(document []byte) (map[string]any, error) {
	const marker = "window.__pinia="
	html := string(document)
	start := strings.Index(html, marker)
	if start < 0 {
		return nil, fmt.Errorf("search page does not contain its state")
	}
	start += len(marker)
	end := strings.Index(html[start:], "</script>")
	if end < 0 {
		return nil, fmt.Errorf("search page state is incomplete")
	}
	source := strings.TrimSpace(html[start : start+end])
	source = strings.TrimSuffix(source, ";")

	runtime := goja.New()
	runtime.SetMaxCallStackSize(256)
	timer := time.AfterFunc(intlSearchEvalLimit, func() {
		runtime.Interrupt("search state evaluation timed out")
	})
	// Parentheses keep an object literal from being parsed as a JavaScript block.
	value, err := runtime.RunString("(" + source + ")")
	timer.Stop()
	if err != nil {
		return nil, fmt.Errorf("evaluate search page state: %w", err)
	}
	root, ok := value.Export().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("search page state has an unexpected shape")
	}
	return root, nil
}

func searchPageRegion(state map[string]any) string {
	return stringValue(mapValue(state["searchResponse"])["ipRegion"])
}

func appendSearchTypeResults(destination *[]any, seen map[string]bool, container map[string]any, allowUntyped bool) {
	appendBangumiSearchRows(destination, seen, container["result"], allowUntyped, 0)
}

func appendBangumiSearchRows(destination *[]any, seen map[string]bool, value any, allowUntyped bool, depth int) {
	if depth > 8 {
		return
	}
	switch typed := value.(type) {
	case []any:
		for _, row := range typed {
			appendBangumiSearchRows(destination, seen, row, allowUntyped, depth+1)
		}
	case map[string]any:
		resultType := strings.ToLower(stringValue(typed["result_type"]))
		if resultType != "" {
			if resultType != "media_bangumi" && resultType != "bangumi" && resultType != "pgc" {
				return
			}
			appendBangumiSearchRows(destination, seen, typed["data"], true, depth+1)
			return
		}
		if allowUntyped && isBangumiSearchCard(typed) {
			if card := normalizeBangumiCard(typed); card != nil {
				appendUniqueCard(destination, seen, card)
			}
			return
		}
		for _, key := range []string{"media_bangumi", "bangumi", "pgc"} {
			if nested, ok := typed[key]; ok {
				appendBangumiSearchRows(destination, seen, nested, true, depth+1)
			}
		}
		for _, key := range []string{"result", "data"} {
			if nested, ok := typed[key]; ok {
				appendBangumiSearchRows(destination, seen, nested, allowUntyped, depth+1)
			}
		}
	}
}

func isBangumiSearchCard(card map[string]any) bool {
	if integerValue(card["season_id"]) > 0 || integerValue(card["pgc_season_id"]) > 0 {
		return true
	}
	kind := strings.ToLower(stringValue(card["type"]))
	return kind == "media_bangumi" || kind == "bangumi"
}

func appendUniqueCard(destination *[]any, seen map[string]bool, card map[string]any) {
	key := stringValue(card["season_id"])
	if key == "" || key == "0" {
		key = stringValue(card["url"])
	}
	if key == "" {
		key = stringValue(card["title"])
	}
	if key == "" || seen[key] {
		return
	}
	seen[key] = true
	*destination = append(*destination, card)
}

func normalizeBangumiCard(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	card := make(map[string]any, len(source)+8)
	for key, value := range source {
		card[key] = normalizeSearchValue(value)
	}
	if strings.TrimSpace(stringValue(card["title"])) == "" {
		return nil
	}
	if card["type"] == nil || stringValue(card["type"]) == "" {
		card["type"] = "media_bangumi"
	}
	if card["cover"] == nil || stringValue(card["cover"]) == "" {
		card["cover"] = card["pic"]
	}
	card["cover"] = secureImageURL(stringValue(card["cover"]))
	if card["url"] == nil || stringValue(card["url"]) == "" {
		card["url"] = card["goto_url"]
	}
	if card["url"] == nil || stringValue(card["url"]) == "" {
		if seasonID := integerValue(card["season_id"]); seasonID > 0 {
			card["url"] = fmt.Sprintf("https://www.bilibili.com/bangumi/play/ss%d", seasonID)
		}
	}
	if card["goto_url"] == nil || stringValue(card["goto_url"]) == "" {
		card["goto_url"] = card["url"]
	}
	if card["areas"] == nil {
		card["areas"] = card["area"]
	}
	if card["styles"] == nil {
		card["styles"] = card["style"]
	}
	return card
}

func normalizeSearchValue(value any) any {
	switch typed := value.(type) {
	case float64:
		if typed == float64(int64(typed)) {
			return int64(typed)
		}
		return typed
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = normalizeSearchValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = normalizeSearchValue(item)
		}
		return out
	default:
		return value
	}
}

func mediaBangumiTotal(response map[string]any) int64 {
	pageInfo := mapValue(response["pageinfo"])
	media := mapValue(pageInfo["media_bangumi"])
	return integerValue(media["numResults"])
}

func searchTypeTotal(response map[string]any) int64 {
	if total := integerValue(response["numResults"]); total > 0 {
		return total
	}
	return integerValue(response["total"])
}

func secureImageURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "//") {
		return "https:" + value
	}
	return strings.Replace(value, "http://", "https://", 1)
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func asSlice(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func integerValue(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed
	default:
		return 0
	}
}
