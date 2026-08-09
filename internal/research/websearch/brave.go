package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 2 << 20

type BraveConfig struct {
	BaseURL       string
	APIKey        string
	Timeout       time.Duration
	MaxResults    int
	SafeSearch    string
	ImportEnabled bool
}

type Brave struct {
	baseURL       string
	apiKey        string
	maxResults    int
	safeSearch    string
	importEnabled bool
	client        *http.Client
}

func NewBrave(cfg BraveConfig) *Brave {
	return &Brave{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"), apiKey: strings.TrimSpace(cfg.APIKey), maxResults: cfg.MaxResults,
		safeSearch: cfg.SafeSearch, importEnabled: cfg.ImportEnabled, client: &http.Client{Timeout: cfg.Timeout},
	}
}

func (b *Brave) Capability() Capability {
	configured := b.apiKey != ""
	details := []string{"HTTPS 搜索", "严格安全搜索", "结果带来源链接"}
	if !configured {
		details = append(details, "尚未配置 Brave Search API Key")
	}
	if configured && !b.importEnabled {
		details = append(details, "搜索结果入库未启用；需确认供应商计划包含存储权")
	}
	return Capability{Configured: configured, Provider: "brave", ImportEnabled: configured && b.importEnabled, MaxResults: b.maxResults, Details: details}
}

func (b *Brave) Search(ctx context.Context, query Query) (Response, error) {
	if b.apiKey == "" {
		return Response{}, &Error{Code: "NOT_CONFIGURED", Message: "实时联网尚未配置 Brave Search API Key"}
	}
	count := query.Count
	if count < 1 || count > b.maxResults {
		count = b.maxResults
	}
	endpoint, err := url.Parse(b.baseURL + "/res/v1/web/search")
	if err != nil {
		return Response{}, &Error{Code: "INVALID_CONFIGURATION", Message: "联网搜索地址配置无效"}
	}
	values := endpoint.Query()
	values.Set("q", query.Text)
	values.Set("count", strconv.Itoa(count))
	values.Set("safe_search", b.safeSearch)
	values.Set("text_decorations", "false")
	values.Set("spellcheck", "true")
	if query.Country != "" {
		values.Set("country", strings.ToUpper(query.Country))
	}
	if query.SearchLang != "" {
		values.Set("search_lang", strings.ToLower(query.SearchLang))
	}
	if query.Freshness != "" {
		values.Set("freshness", query.Freshness)
	}
	endpoint.RawQuery = values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Response{}, &Error{Code: "INVALID_REQUEST", Message: "无法创建联网搜索请求"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.apiKey)
	resp, err := b.client.Do(req)
	if err != nil {
		return Response{}, &Error{Code: "UPSTREAM_UNAVAILABLE", Message: "联网搜索超时或网络不可用", Retryable: true}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Response{}, classifyStatus(resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return Response{}, &Error{Code: "INVALID_RESPONSE", Message: "联网搜索响应无法读取"}
	}
	var payload struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
				Age         string `json:"age"`
				PageAge     string `json:"page_age"`
				Language    string `json:"language"`
				Profile     struct {
					LongName string `json:"long_name"`
				} `json:"profile"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Response{}, &Error{Code: "INVALID_RESPONSE", Message: "联网搜索返回了无法解析的数据"}
	}
	results := make([]Result, 0, len(payload.Web.Results))
	for _, item := range payload.Web.Results {
		parsed, parseErr := url.Parse(strings.TrimSpace(item.URL))
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			continue
		}
		publisher := strings.TrimSpace(item.Profile.LongName)
		if publisher == "" {
			publisher = strings.ToLower(parsed.Hostname())
		}
		publishedAt := strings.TrimSpace(item.PageAge)
		if publishedAt == "" {
			publishedAt = strings.TrimSpace(item.Age)
		}
		results = append(results, Result{Title: strings.TrimSpace(item.Title), URL: parsed.String(), Description: strings.TrimSpace(item.Description), Publisher: publisher, PublishedAt: publishedAt, Language: item.Language})
	}
	return Response{Provider: "brave", Results: results, SearchedAt: time.Now().UTC()}, nil
}

func classifyStatus(status int) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &Error{Code: "UNAUTHORIZED", Message: "Brave Search API Key 无效或无权访问"}
	case http.StatusTooManyRequests:
		return &Error{Code: "RATE_LIMITED", Message: "联网搜索达到供应商限流，请稍后重试", Retryable: true}
	default:
		if status >= 500 {
			return &Error{Code: "UPSTREAM_UNAVAILABLE", Message: "联网搜索服务暂时不可用", Retryable: true}
		}
		return &Error{Code: "UPSTREAM_ERROR", Message: fmt.Sprintf("联网搜索请求失败（HTTP %d）", status)}
	}
}
