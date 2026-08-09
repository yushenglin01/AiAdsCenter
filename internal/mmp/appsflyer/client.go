package appsflyer

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	mmpprovider "github.com/example/adnova/internal/mmp/provider"
	"github.com/example/adnova/pkg/provider"
	"github.com/shopspring/decimal"
)

const (
	maxRows         = 200000
	maxResponseSize = 50 << 20
)

type Config struct {
	BaseURL        string
	Token          string
	Timeout        time.Duration
	MaxRetries     int
	PurchaseEvents []string
}

type Client struct {
	baseURL        string
	token          string
	maxRetries     int
	purchaseEvents []string
	httpClient     *http.Client
	wait           func(context.Context, time.Duration) error
}

type FetchInput = mmpprovider.FetchInput
type FetchResult = mmpprovider.FetchResult
type ProviderError = mmpprovider.Error

func New(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"), token: strings.TrimSpace(cfg.Token), maxRetries: cfg.MaxRetries,
		purchaseEvents: append([]string(nil), cfg.PurchaseEvents...),
		httpClient:     &http.Client{Timeout: cfg.Timeout}, wait: waitContext,
	}
}

func (c *Client) Configured() bool { return c.token != "" }

func (c *Client) Fetch(ctx context.Context, input FetchInput) (*FetchResult, error) {
	if !c.Configured() {
		return nil, &ProviderError{Code: "NOT_CONFIGURED", Message: "AppsFlyer API Token 尚未配置"}
	}
	if strings.TrimSpace(input.AppID) == "" {
		return nil, &ProviderError{Code: "INVALID_REQUEST", Message: "AppsFlyer App ID 不能为空"}
	}
	fetchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	type reportResult struct {
		kind string
		rows []map[string]string
		err  error
	}
	results := make(chan reportResult, 2)
	var wg sync.WaitGroup
	for _, report := range []string{"installs_report", "in_app_events_report"} {
		wg.Add(1)
		go func(kind string) {
			defer wg.Done()
			rows, err := c.fetchReport(fetchCtx, input, kind)
			results <- reportResult{kind: kind, rows: rows, err: err}
		}(report)
	}
	go func() { wg.Wait(); close(results) }()
	reports := map[string][]map[string]string{}
	for result := range results {
		if result.err != nil {
			cancel()
			return nil, result.err
		}
		reports[result.kind] = result.rows
	}
	return normalize(reports["installs_report"], reports["in_app_events_report"])
}

func (c *Client) fetchReport(ctx context.Context, input FetchInput, report string) ([]map[string]string, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/raw-data/export/app/" + url.PathEscape(input.AppID) + "/" + report + "/v5")
	if err != nil {
		return nil, &ProviderError{Code: "INVALID_REQUEST", Message: "AppsFlyer 请求地址无效"}
	}
	query := endpoint.Query()
	query.Set("from", input.From.Format("2006-01-02"))
	query.Set("to", input.To.Format("2006-01-02"))
	query.Set("currency", "USD")
	query.Set("maximum_rows", strconv.Itoa(maxRows))
	if report == "in_app_events_report" {
		query.Set("event_name", strings.Join(c.purchaseEvents, ","))
	}
	endpoint.RawQuery = query.Encode()

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if requestErr != nil {
			return nil, &ProviderError{Code: "INVALID_REQUEST", Message: "无法创建 AppsFlyer 请求"}
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "text/csv")
		resp, requestErr := c.httpClient.Do(req)
		if requestErr != nil {
			if attempt < c.maxRetries && ctx.Err() == nil {
				if err := c.wait(ctx, backoff(attempt)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, &ProviderError{Code: "UPSTREAM_UNAVAILABLE", Message: "AppsFlyer 请求超时或网络不可用"}
		}
		rows, responseErr, retry := decodeResponse(resp)
		if responseErr == nil {
			return rows, nil
		}
		if retry && attempt < c.maxRetries {
			if err := c.wait(ctx, retryDelay(resp, attempt)); err != nil {
				return nil, err
			}
			continue
		}
		return nil, responseErr
	}
	return nil, &ProviderError{Code: "UPSTREAM_UNAVAILABLE", Message: "AppsFlyer 请求失败"}
}

func decodeResponse(resp *http.Response) ([]map[string]string, *ProviderError, bool) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		code, message, retry := classifyStatus(resp.StatusCode, string(body))
		return nil, &ProviderError{Code: code, Message: message}, retry
	}
	limited := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, &ProviderError{Code: "INVALID_RESPONSE", Message: "读取 AppsFlyer 响应失败"}, false
	}
	if len(body) > maxResponseSize {
		return nil, &ProviderError{Code: "RESPONSE_TOO_LARGE", Message: "AppsFlyer 响应超过 50MB，请缩小日期范围"}, false
	}
	rows, err := decodeCSV(strings.NewReader(string(body)))
	if err != nil {
		return nil, &ProviderError{Code: "INVALID_RESPONSE", Message: "AppsFlyer 返回了无法解析的 CSV"}, false
	}
	if len(rows) >= maxRows {
		return nil, &ProviderError{Code: "ROW_LIMIT_REACHED", Message: "AppsFlyer 返回达到 20 万行上限，请缩小日期范围"}, false
	}
	return rows, nil, false
}

func decodeCSV(reader io.Reader) ([]map[string]string, error) {
	r := csv.NewReader(reader)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err == io.EOF {
		return []map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	for i := range header {
		header[i] = normalizeHeader(header[i])
	}
	rows := make([]map[string]string, 0)
	for {
		values, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(values) != len(header) {
			return nil, fmt.Errorf("unexpected column count")
		}
		row := make(map[string]string, len(header))
		for i := range header {
			row[header[i]] = strings.TrimSpace(values[i])
		}
		rows = append(rows, row)
	}
	return rows, nil
}

type metric struct {
	date, campaign, country string
	installs                int64
	revenue                 decimal.Decimal
	payers                  map[string]struct{}
}

func normalize(installs, events []map[string]string) (*FetchResult, error) {
	metrics := map[string]*metric{}
	skipped, missingPayer := 0, 0
	add := func(row map[string]string, dateField string) (*metric, bool, error) {
		campaign := first(row, "campaign_id", "campaign")
		if campaign == "" {
			skipped++
			return nil, false, nil
		}
		date, err := parseDate(first(row, dateField, "event_time", "install_time"))
		if err != nil {
			return nil, false, &ProviderError{Code: "INVALID_RESPONSE", Message: "AppsFlyer 响应包含无效日期"}
		}
		country := strings.ToUpper(first(row, "country_code", "country"))
		if country == "" {
			country = "ZZ"
		}
		key := strings.Join([]string{date, campaign, country}, "\x00")
		item := metrics[key]
		if item == nil {
			item = &metric{date: date, campaign: campaign, country: country, revenue: decimal.Zero, payers: map[string]struct{}{}}
			metrics[key] = item
		}
		return item, true, nil
	}
	for _, row := range installs {
		item, ok, err := add(row, "install_time")
		if err != nil {
			return nil, err
		}
		if ok {
			item.installs++
		}
	}
	for _, row := range events {
		item, ok, err := add(row, "event_time")
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		revenueText := first(row, "event_revenue_usd")
		if revenueText == "" && first(row, "event_revenue") != "" {
			if strings.ToUpper(first(row, "event_revenue_currency")) != "USD" {
				return nil, &ProviderError{Code: "INVALID_RESPONSE", Message: "AppsFlyer 付费事件缺少可验证的 USD 收入字段"}
			}
			revenueText = first(row, "event_revenue")
		}
		if revenueText != "" {
			revenue, err := decimal.NewFromString(revenueText)
			if err != nil || revenue.IsNegative() {
				return nil, &ProviderError{Code: "INVALID_RESPONSE", Message: "AppsFlyer 响应包含无效或负数收入"}
			}
			item.revenue = item.revenue.Add(revenue)
		}
		payerID := first(row, "appsflyer_id", "customer_user_id")
		if payerID == "" {
			missingPayer++
		} else {
			item.payers[payerID] = struct{}{}
		}
	}
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	records := make([]provider.Record, 0, len(keys))
	for _, key := range keys {
		item := metrics[key]
		records = append(records, provider.Record{
			"date": item.date, "campaign_external_id": item.campaign, "country": item.country, "currency": "USD",
			"installs": strconv.FormatInt(item.installs, 10), "activations": strconv.FormatInt(item.installs, 10),
			"payers": strconv.Itoa(len(item.payers)), "revenue": item.revenue.StringFixed(6),
		})
	}
	warnings := make([]string, 0, 2)
	if skipped > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 行缺少 campaign_id，已跳过", skipped))
	}
	if missingPayer > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个付费事件缺少用户标识，未计入付费人数", missingPayer))
	}
	if len(installs)+len(events) > 0 && len(records) == 0 {
		return nil, &ProviderError{Code: "NO_MAPPABLE_ROWS", Message: "AppsFlyer 返回记录均缺少 campaign_id，未替换现有指标"}
	}
	return &FetchResult{Records: records, SourceRows: len(installs) + len(events), SkippedRows: skipped, WarningMessage: strings.Join(warnings, "；")}, nil
}

func first(row map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(row[key]); value != "" {
			return value
		}
	}
	return ""
}

func parseDate(value string) (string, error) {
	if len(value) < 10 {
		return "", fmt.Errorf("short date")
	}
	parsed, err := time.Parse("2006-01-02", value[:10])
	if err != nil {
		return "", err
	}
	return parsed.Format("2006-01-02"), nil
}

func normalizeHeader(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func classifyStatus(status int, responseBody string) (string, string, bool) {
	if status == http.StatusBadRequest {
		normalized := strings.ToLower(responseBody)
		if strings.Contains(normalized, "calllimit") || strings.Contains(normalized, "rate limit") {
			return "RATE_LIMITED", "AppsFlyer 请求达到限流", true
		}
	}
	switch status {
	case http.StatusBadRequest:
		return "INVALID_REQUEST", "AppsFlyer 拒绝了查询参数", false
	case http.StatusUnauthorized:
		return "UNAUTHORIZED", "AppsFlyer Token 无效或账号已停用", false
	case http.StatusForbidden:
		return "FORBIDDEN", "AppsFlyer Token 无权读取该应用", false
	case http.StatusNotFound:
		return "NOT_FOUND", "AppsFlyer App ID 不存在或 Token 不匹配", false
	case http.StatusTooManyRequests:
		return "RATE_LIMITED", "AppsFlyer 请求达到限流", true
	default:
		if status >= 500 {
			return "UPSTREAM_UNAVAILABLE", "AppsFlyer 服务暂时不可用", true
		}
		return "UPSTREAM_ERROR", "AppsFlyer 返回异常状态", false
	}
}

func backoff(attempt int) time.Duration { return time.Duration(1<<attempt) * 200 * time.Millisecond }
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 && seconds <= 5 {
		return time.Duration(seconds) * time.Second
	}
	return backoff(attempt)
}
func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
