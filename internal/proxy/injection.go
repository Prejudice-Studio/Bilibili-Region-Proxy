package proxy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────
// Search result injection
// ─────────────────────────────────────────────────────────────────────────

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case float32:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	case json.Number:
		f, err := t.Float64()
		return err == nil && f != 0
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	default:
		return true
	}
}

func firstTruthy(entry map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := entry[k]; ok && truthy(v) {
			return v
		}
	}
	return nil
}

func decodeHTTPLinkValue(value any, allowBareHost bool) string {
	if !truthy(value) {
		return ""
	}
	link := strings.TrimSpace(fmt.Sprint(value))
	if strings.HasPrefix(link, "//") {
		link = "https:" + link
	} else if !strings.Contains(link, ":") {
		switch {
		case strings.HasPrefix(link, "/"):
			link = "https://www.bilibili.com" + link
		case allowBareHost:
			link = "https://" + link
		default:
			return ""
		}
	}
	parsed, err := url.Parse(link)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
		return ""
	}
	return link
}

func toInt(v any, def int64) int64 {
	switch t := v.(type) {
	case nil:
		return def
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case bool:
		if t {
			return 1
		}
		return 0
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return def
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
		return def
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return n
		}
		if f, err := t.Float64(); err == nil {
			return int64(f)
		}
		return def
	default:
		return def
	}
}

func toFloat(v any, def float64) float64 {
	switch t := v.(type) {
	case nil:
		return def
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return f
		}
		return def
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f
		}
		return def
	default:
		return def
	}
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit])
}

func textValue(v any, limit int) string {
	switch v.(type) {
	case nil, map[string]any, []any:
		return ""
	}
	return truncateRunes(strings.TrimSpace(fmt.Sprint(v)), limit)
}

func listValue(v any) []any {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	return list
}

func limitList(list []any, limit int) []any {
	if limit >= 0 && len(list) > limit {
		return list[:limit]
	}
	return list
}

func toAnySlice(items []map[string]any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

var seasonTypeNames = map[int64]string{
	1: "番剧", 2: "电影", 3: "纪录片", 4: "国创", 5: "电视剧", 7: "综艺",
}

func configuredSearchInjections(cfg *proxyConfig) ([]map[string]any, string) {
	if cfg == nil {
		c := loadProxyConfig()
		cfg = &c
	}
	if !cfg.injectionEnabled {
		return nil, "tail"
	}
	position := cfg.injectionPosition
	if position != "head" {
		position = "tail"
	}

	entries := cfg.injectionItems
	if len(entries) > 100 {
		entries = entries[:100]
	}
	results := make([]map[string]any, 0, len(entries))
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		title := textValue(entry["title"], 200)
		link := decodeHTTPLinkValue(firstTruthy(entry, "link", "url", "goto_url", "uri"), true)
		image := textValue(firstTruthy(entry, "image", "image_url", "cover"), 2048)
		if title == "" || link == "" || len(link) > 2048 {
			continue
		}

		seasonID := toInt(firstTruthy(entry, "season_id", "pgc_season_id"), 0)

		mediaScore, _ := entry["media_score"].(map[string]any)
		if mediaScore == nil {
			mediaScore = map[string]any{}
		}
		getScoreKey := func(key string) any {
			if v, ok := mediaScore[key]; ok {
				return v
			}
			return entry["media_score."+key]
		}
		rating := getScoreKey("score")
		if rating == nil {
			rating = entry["rating"]
		}
		scoreCount := getScoreKey("user_count")
		if scoreCount == nil {
			scoreCount = entry["rating_count"]
		}
		if scoreCount == nil {
			scoreCount = entry["vote"]
		}
		ratingValue := toFloat(rating, 0)
		var mediaScoreValue any
		if ratingValue != 0 || truthy(scoreCount) {
			mediaScoreValue = map[string]any{"score": ratingValue, "user_count": toInt(scoreCount, 0)}
		}

		episodeValues := listValue(entry["eps"])
		if len(episodeValues) == 0 {
			episodeValues = listValue(entry["episodes"])
		}
		episodeValues = limitList(episodeValues, 30)

		requestedType := textValue(entry["type"], 40)
		seasonDefault := int64(1)
		if requestedType == "media_ft" {
			seasonDefault = 2
		}
		seasonType := toInt(entry["season_type"], seasonDefault)
		if seasonType == 0 {
			seasonType = seasonDefault
		}
		seasonTypeName := textValue(entry["season_type_name"], 80)
		if seasonTypeName == "" {
			seasonTypeName = seasonTypeNames[seasonType]
			if seasonTypeName == "" {
				seasonTypeName = "番剧"
			}
		}
		allNet, _ := entry["all_net"].(map[string]any)
		if allNet == nil {
			allNet = map[string]any{}
		}
		allNetName := firstTruthy(entry, "all_net_name")
		if allNetName == nil {
			allNetName = allNet["name"]
		}
		allNetIcon := firstTruthy(entry, "all_net_icon")
		if allNetIcon == nil {
			allNetIcon = allNet["icon"]
		}
		allNetURL := firstTruthy(entry, "all_net_url")
		if allNetURL == nil {
			allNetURL = allNet["url"]
		}

		resultType := requestedType
		if resultType == "" {
			if seasonType == 1 || seasonType == 4 {
				resultType = "media_bangumi"
			} else {
				resultType = "media_ft"
			}
		}

		mediaType := toInt(entry["media_type"], 1)
		if mediaType == 0 {
			mediaType = 1
		}

		desc := firstTruthy(entry, "desc", "description", "prompt")
		description := firstTruthy(entry, "description", "desc", "prompt")
		prompt := firstTruthy(entry, "prompt", "desc", "description")
		outName := firstTruthy(entry, "out_name", "all_net_name")
		if outName == nil {
			outName = allNet["name"]
		}
		outIcon := firstTruthy(entry, "out_icon", "all_net_icon")
		if outIcon == nil {
			outIcon = allNet["icon"]
		}
		outURL := firstTruthy(entry, "out_url", "all_net_url")
		if outURL == nil {
			outURL = allNet["url"]
		}

		watchButton, _ := entry["watch_button"].(map[string]any)
		checkMore, _ := entry["check_more"].(map[string]any)
		followButton, _ := entry["follow_button"].(map[string]any)
		styleLabel, _ := entry["style_label"].(map[string]any)

		result := map[string]any{
			"resolver_injected":         true,
			"resolver_injection_source": "Bilibili-Region-Proxy",
			"type":                      resultType,
			"title":                     title,
			"org_title":                 textValue(firstTruthy(entry, "org_title", "original_title"), 200),
			"cover":                     image,
			"media_type":                mediaType,
			"areas":                     textValue(firstTruthy(entry, "areas", "area"), 300),
			"area":                      textValue(firstTruthy(entry, "area", "areas"), 300),
			"styles":                    textValue(firstTruthy(entry, "styles", "style"), 500),
			"style":                     textValue(firstTruthy(entry, "style", "styles"), 500),
			"cv":                        textValue(entry["cv"], 1000),
			"staff":                     textValue(entry["staff"], 1000),
			"play_state":                toInt(entry["play_state"], 0),
			"goto_url":                  link,
			"url":                       link,
			"link":                      link,
			"uri":                       link,
			"param":                     textValue(entry["param"], 2048),
			"goto":                      textValue(entry["goto"], 120),
			"linktype":                  textValue(entry["linktype"], 120),
			"trackid":                   textValue(entry["trackid"], 200),
			"desc":                      textValue(desc, 5000),
			"description":               textValue(description, 5000),
			"prompt":                    textValue(prompt, 5000),
			"pubtime":                   toInt(entry["pubtime"], 0),
			"fix_pubtime_str":           textValue(entry["fix_pubtime_str"], 200),
			"media_score":               mediaScoreValue,
			"rating":                    ratingValue,
			"season_type":               seasonType,
			"season_type_name":          seasonTypeName,
			"season_id":                 seasonID,
			"pgc_season_id":             seasonID,
			"media_id":                  toInt(entry["media_id"], 0),
			"media_mode":                toInt(entry["media_mode"], 0),
			"is_avid":                   truthy(entry["is_avid"]),
			"corner":                    toInt(entry["corner"], 0),
			"hit_columns":               listValue(entry["hit_columns"]),
			"all_net_name":              textValue(allNetName, 200),
			"all_net_icon":              textValue(allNetIcon, 2048),
			"all_net_url":               textValue(allNetURL, 2048),
			"angle_title":               textValue(entry["angle_title"], 200),
			"angle_color":               toInt(entry["angle_color"], 0),
			"display_info":              listValue(entry["display_info"]),
			"hit_epids":                 textValue(entry["hit_epids"], 1000),
			"selection_style":           textValue(entry["selection_style"], 80),
			"ep_size":                   toInt(entry["ep_size"], int64(len(episodeValues))),
			"eps":                       episodeValues,
			"episodes":                  episodeValues,
			"episodes_new":              limitList(listValue(entry["episodes_new"]), 30),
			"badges":                    listValue(entry["badges"]),
			"badges_v2":                 listValue(entry["badges_v2"]),
			"label":                     textValue(entry["label"], 200),
			"is_out":                    toInt(entry["is_out"], 0),
			"styles_v2":                 textValue(entry["styles_v2"], 500),
			"style_label":               styleLabel,
			"target":                    textValue(entry["target"], 2048),
			"out_name":                  textValue(outName, 200),
			"out_icon":                  textValue(outIcon, 2048),
			"out_url":                   textValue(outURL, 2048),
			"button_text":               textValue(entry["button_text"], 120),
			"watch_button":              watchButton,
			"check_more":                checkMore,
			"follow_button":             followButton,
			"is_atten":                  toInt(firstTruthy(entry, "is_atten", "is_follow"), 0),
			"is_follow":                 toInt(entry["is_follow"], 0),
			"is_selection":              toInt(entry["is_selection"], 0),
			"num_episodes":              len(episodeValues),
		}

		if seasonID != 0 {
			result["season_id"] = seasonID
			result["pgc_season_id"] = seasonID
		} else {
			delete(result, "season_id")
			delete(result, "pgc_season_id")
		}
		if toInt(result["media_id"], 0) == 0 {
			delete(result, "media_id")
		}

		results = append(results, result)
	}
	return results, position
}

func isZeroCode(v any) bool {
	switch t := v.(type) {
	case float64:
		return t == 0
	case int:
		return t == 0
	case int64:
		return t == 0
	case string:
		return t == "0"
	default:
		return false
	}
}

func injectSearchResults(response map[string]any, cfg *proxyConfig) map[string]any {
	additions, position := configuredSearchInjections(cfg)
	if len(additions) == 0 || response == nil {
		return response
	}
	if code, ok := response["code"]; ok && !isZeroCode(code) {
		return response
	}
	data := response["data"]
	if data == nil {
		if isZeroCode(response["code"]) {
			empty := map[string]any{}
			response["data"] = empty
			data = empty
		}
	}
	dm, ok := data.(map[string]any)
	if !ok {
		return response
	}

	resultValue, hasResult := dm["result"]
	itemsValue, hasItems := dm["items"]
	var key string
	switch {
	case isList(resultValue) || (hasResult && resultValue == nil):
		key = "result"
	case isList(itemsValue) || (hasItems && itemsValue == nil):
		key = "items"
	case !hasResult && !hasItems:
		key = "result"
	default:
		return response
	}

	var current []any
	if raw := dm[key]; raw != nil {
		list, ok := raw.([]any)
		if !ok {
			return response
		}
		current = list
	}

	addAny := toAnySlice(additions)
	merged := make([]any, 0, len(current)+len(addAny))
	if position == "head" {
		merged = append(merged, addAny...)
		merged = append(merged, current...)
	} else {
		merged = append(merged, current...)
		merged = append(merged, addAny...)
	}
	dm[key] = merged
	// numResults describes Bilibili's actual matches and excludes injected entries.
	return response
}

func isList(v any) bool {
	_, ok := v.([]any)
	return ok
}

func isFirstSearchPage(query url.Values) bool {
	if truthy(query.Get("resolver_cursor")) || truthy(query.Get("cursor")) || truthy(query.Get("next")) {
		return false
	}
	for _, name := range []string{"page", "pn"} {
		value := query.Get(name)
		if value == "" {
			continue
		}
		n, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		return n == 1
	}
	return true
}
