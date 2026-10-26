package proxy

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestIntlSearchQueryForwardsClientParameters(t *testing.T) {
	input := url.Values{
		"keyword":         {"旧关键词"},
		"search_type":     {"media_bangumi"},
		"page_size":       {"30"},
		"access_key":      {"account-token"},
		"resolver_mode":   {"INTL"},
		"resolver_cursor": {"local-cursor"},
	}
	got := intlSearchQuery(input, "我的英雄学院", 2)
	if got.Get("keyword") != "我的英雄学院" || got.Get("category") != "bangumi" || got.Get("page") != "2" {
		t.Fatalf("search parameters were not overridden correctly: %v", got)
	}
	if got.Get("search_type") != "media_bangumi" || got.Get("page_size") != "30" || got.Get("access_key") != "account-token" {
		t.Fatalf("client parameters were not retained: %v", got)
	}
	if got.Get("resolver_mode") != "" || got.Get("resolver_cursor") != "" {
		t.Fatalf("resolver controls leaked upstream: %v", got)
	}
}

func TestIntlSearchHeadersForwardAuthentication(t *testing.T) {
	input := http.Header{
		"Cookie":          {"SESSDATA=account-cookie"},
		"Authorization":   {"Bearer account-token"},
		"X-Bili-Mid":      {"123"},
		"resolver_mode":   {"INTL"},
		"Accept-Encoding": {"gzip"},
	}
	got := intlSearchHeaders(input)
	if got.Get("Cookie") != "SESSDATA=account-cookie" || got.Get("Authorization") != "Bearer account-token" || got.Get("X-Bili-Mid") != "123" {
		t.Fatalf("authentication/device headers were not forwarded: %v", got)
	}
	if got.Get("resolver_mode") != "" || got.Get("Accept-Encoding") != "identity" {
		t.Fatalf("internal or unsafe headers were not handled: %v", got)
	}
	if strings.TrimSpace(got.Get("User-Agent")) == "" {
		t.Fatal("default User-Agent was not added")
	}
}

func TestPreserveSearchQueryOnRedirect(t *testing.T) {
	previousURL, err := url.Parse("https://search.bilibili.com/zh-Hans/bangumi?keyword=%E6%88%91%E7%9A%84%E8%8B%B1%E9%9B%84%E5%AD%A6%E9%99%A2&category=bangumi&page=1")
	if err != nil {
		t.Fatal(err)
	}
	next, err := http.NewRequest(http.MethodGet, "https://search.bilibili.com/bangumi", nil)
	if err != nil {
		t.Fatal(err)
	}
	previous := &http.Request{URL: previousURL, Header: make(http.Header)}
	previous.Header.Set("Accept-Language", "zh-Hans-SG")
	if err := preserveSearchQueryOnRedirect(next, []*http.Request{previous}); err != nil {
		t.Fatalf("preserveSearchQueryOnRedirect returned error: %v", err)
	}
	if got, want := next.URL.RawQuery, previousURL.RawQuery; got != want {
		t.Fatalf("redirect query = %q, want %q", got, want)
	}
	if got := next.Header.Get("Accept-Language"); got != "zh-Hans-SG" {
		t.Fatalf("redirect language = %q, want zh-Hans-SG", got)
	}
}

func TestIntlBangumiResponseReadsObjectResult(t *testing.T) {
	document := []byte(`window.__pinia={"searchResponse":{"ipRegion":"SG","searchAllResponse":{"pageinfo":{"media_bangumi":{"numResults":1,"pages":1}},"result":[]}},"searchTypeResponse":{"searchTypeResponse":{"numResults":1,"numPages":1,"result":{"media_bangumi":{"data":[{"title":"我的英雄学院：两位英雄","season_id":274813,"cover":"//i0.hdslb.com/test.jpg"}]}}}}};</script>`)
	response, region, err := intlBangumiResponse(document, 1, 20)
	if err != nil {
		t.Fatalf("intlBangumiResponse returned error: %v", err)
	}
	if region != "SG" {
		t.Fatalf("region = %q, want SG", region)
	}
	data := mapValue(response["data"])
	rows := asSlice(data["result"])
	if len(rows) != 1 {
		t.Fatalf("result count = %d, want 1", len(rows))
	}
	card := mapValue(rows[0])
	if got := stringValue(card["title"]); got != "我的英雄学院：两位英雄" {
		t.Fatalf("title = %q", got)
	}
	if got := stringValue(card["url"]); got != "https://www.bilibili.com/bangumi/play/ss274813" {
		t.Fatalf("url = %q", got)
	}
}

func TestSearchAllFallbackKeepsOnlyBangumiRows(t *testing.T) {
	searchAll := map[string]any{
		"result": []any{
			map[string]any{"result_type": "video", "data": []any{map[string]any{"title": "video", "season_id": 1}}},
			map[string]any{"result_type": "media_bangumi", "data": []any{map[string]any{"title": "bangumi", "season_id": 2}}},
		},
	}
	rows := make([]any, 0)
	appendSearchTypeResults(&rows, make(map[string]bool), searchAll, false)
	if len(rows) != 1 {
		t.Fatalf("result count = %d, want 1", len(rows))
	}
	card := mapValue(rows[0])
	if got := card["title"]; got != "bangumi" {
		t.Fatalf("title = %v, want bangumi", got)
	}
}
