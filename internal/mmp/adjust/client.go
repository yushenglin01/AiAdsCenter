package adjust

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	mmpprovider "github.com/example/adnova/internal/mmp/provider"
	dataprovider "github.com/example/adnova/pkg/provider"
	"github.com/shopspring/decimal"
)

const maxResponseSize = 50 << 20

type Config struct {
	BaseURL          string
	Token            string
	ActivationMetric string
	PayerMetric      string
	RevenueMetric    string
	Timeout          time.Duration
	MaxRetries       int
}

type Client struct {
	baseURL          string
	token            string
	activationMetric string
	payerMetric      string
	revenueMetric    string
	maxRetries       int
	httpClient       *http.Client
	wait             func(context.Context, time.Duration) error
}

func New(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"), token: strings.TrimSpace(cfg.Token),
		activationMetric: strings.TrimSpace(cfg.ActivationMetric), payerMetric: strings.TrimSpace(cfg.PayerMetric),
		revenueMetric: strings.TrimSpace(cfg.RevenueMetric), maxRetries: cfg.MaxRetries,
		httpClient: &http.Client{Timeout: cfg.Timeout}, wait: waitContext,
	}
}

func (c *Client) Configured() bool {
	return c.token != "" && c.activationMetric != "" && c.payerMetric != "" && c.revenueMetric != ""
}

func (c *Client) Fetch(ctx context.Context, input mmpprovider.FetchInput) (*mmpprovider.FetchResult, error) {
	if !c.Configured() {
		return nil, &mmpprovider.Error{Code: "NOT_CONFIGURED", Message: "Adjust API Token 或事件指标映射尚未配置"}
	}
	if strings.TrimSpace(input.AppID) == "" {
		return nil, &mmpprovider.Error{Code: "INVALID_REQUEST", Message: "Adjust App Token 不能为空"}
	}
	endpoint, err := url.Parse(c.baseURL + "/reports-service/report")
	if err != nil {
		return nil, &mmpprovider.Error{Code: "INVALID_REQUEST", Message: "Adjust 请求地址无效"}
	}
	query := endpoint.Query()
	query.Set("app_token__in", input.AppID)
	query.Set("date_period", input.From.Format("2006-01-02")+":"+input.To.Format("2006-01-02"))
	query.Set("dimensions", "day,campaign_id_network,country_code,currency_code")
	query.Set("metrics", strings.Join([]string{"installs", c.activationMetric, c.payerMetric, c.revenueMetric}, ","))
	query.Set("currency", "USD")
	query.Set("utc_offset", "+00:00")
	query.Set("format_dates", "false")
	query.Set("attribution_source", "first")
	endpoint.RawQuery = query.Encode()

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if requestErr != nil {
			return nil, &mmpprovider.Error{Code: "INVALID_REQUEST", Message: "无法创建 Adjust 请求"}
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")
		resp, requestErr := c.httpClient.Do(req)
		if requestErr != nil {
			if attempt < c.maxRetries && ctx.Err() == nil {
				if err := c.wait(ctx, backoff(attempt)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, &mmpprovider.Error{Code: "UPSTREAM_UNAVAILABLE", Message: "Adjust 请求超时或网络不可用"}
		}
		result, responseErr, retry := c.decodeResponse(resp)
		if responseErr == nil {
			return result, nil
		}
		if retry && attempt < c.maxRetries {
			if err := c.wait(ctx, backoff(attempt)); err != nil {
				return nil, err
			}
			continue
		}
		return nil, responseErr
	}
	return nil, &mmpprovider.Error{Code: "UPSTREAM_UNAVAILABLE", Message: "Adjust 请求失败"}
}

type reportResponse struct {
	Rows       []map[string]any `json:"rows"`
	Warnings   []any            `json:"warnings"`
	Pagination any              `json:"pagination"`
}

func (c *Client) decodeResponse(resp *http.Response) (*mmpprovider.FetchResult, *mmpprovider.Error, bool) {
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return &mmpprovider.FetchResult{Records: []dataprovider.Record{}}, nil, false
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		code, message, retry := classifyStatus(resp.StatusCode)
		return nil, &mmpprovider.Error{Code: code, Message: message}, retry
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "读取 Adjust 响应失败"}, false
	}
	if len(body) > maxResponseSize {
		return nil, &mmpprovider.Error{Code: "RESPONSE_TOO_LARGE", Message: "Adjust 响应超过 50MB，请缩小日期范围"}, false
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	var report reportResponse
	if err := decoder.Decode(&report); err != nil {
		return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "Adjust 返回了无法解析的 JSON"}, false
	}
	if report.Pagination != nil {
		return nil, &mmpprovider.Error{Code: "ROW_LIMIT_REACHED", Message: "Adjust 报表需要分页，请缩小日期范围"}, false
	}
	result, normalizeErr := c.normalize(report.Rows)
	if normalizeErr != nil {
		return nil, normalizeErr, false
	}
	if len(report.Rows) > 0 && len(result.Records) == 0 {
		return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "Adjust 报表没有可映射的 campaign_id，已保留现有指标"}, false
	}
	if len(report.Warnings) > 0 {
		result.WarningMessage = fmt.Sprintf("Adjust 返回 %d 条报表警告", len(report.Warnings))
	}
	return result, nil, false
}

type metric struct {
	date, campaign, country, currency string
	installs, activations, payers     int64
	revenue                           decimal.Decimal
}

func (c *Client) normalize(rows []map[string]any) (*mmpprovider.FetchResult, *mmpprovider.Error) {
	aggregated := make(map[string]*metric)
	skipped := 0
	for _, row := range rows {
		date := text(row["day"])
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "Adjust 响应包含无效日期"}
		}
		campaign := strings.TrimSpace(text(row["campaign_id_network"]))
		if campaign == "" || strings.EqualFold(campaign, "unknown") {
			skipped++
			continue
		}
		country := strings.ToUpper(strings.TrimSpace(text(row["country_code"])))
		if len(country) != 2 {
			country = "ZZ"
		}
		currency := strings.ToUpper(strings.TrimSpace(text(row["currency_code"])))
		if len(currency) != 3 {
			currency = "USD"
		}
		installs, err := nonNegativeInt(row["installs"])
		if err != nil {
			return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "Adjust installs 指标无效"}
		}
		activations, err := nonNegativeInt(row[c.activationMetric])
		if err != nil {
			return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "Adjust activation 指标无效"}
		}
		payers, err := nonNegativeInt(row[c.payerMetric])
		if err != nil {
			return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "Adjust payer 指标无效"}
		}
		revenue, err := nonNegativeDecimal(row[c.revenueMetric])
		if err != nil {
			return nil, &mmpprovider.Error{Code: "INVALID_RESPONSE", Message: "Adjust revenue 指标无效"}
		}
		key := strings.Join([]string{date, campaign, country, currency}, "\x00")
		item := aggregated[key]
		if item == nil {
			item = &metric{date: date, campaign: campaign, country: country, currency: currency, revenue: decimal.Zero}
			aggregated[key] = item
		}
		item.installs += installs
		item.activations += activations
		item.payers += payers
		item.revenue = item.revenue.Add(revenue)
	}
	keys := make([]string, 0, len(aggregated))
	for key := range aggregated {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	records := make([]dataprovider.Record, 0, len(keys))
	for _, key := range keys {
		item := aggregated[key]
		records = append(records, dataprovider.Record{
			"date": item.date, "campaign_external_id": item.campaign, "country": item.country, "currency": item.currency,
			"installs": strconv.FormatInt(item.installs, 10), "activations": strconv.FormatInt(item.activations, 10),
			"payers": strconv.FormatInt(item.payers, 10), "revenue": item.revenue.StringFixed(6),
		})
	}
	return &mmpprovider.FetchResult{Records: records, SourceRows: len(rows), SkippedRows: skipped}, nil
}

func text(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func nonNegativeInt(value any) (int64, error) {
	textValue := strings.TrimSpace(text(value))
	if textValue == "" || textValue == "-" {
		return 0, nil
	}
	parsed, err := decimal.NewFromString(textValue)
	if err != nil || parsed.IsNegative() || !parsed.Equal(parsed.Truncate(0)) {
		return 0, fmt.Errorf("invalid non-negative integer")
	}
	return parsed.IntPart(), nil
}

func nonNegativeDecimal(value any) (decimal.Decimal, error) {
	textValue := strings.TrimSpace(text(value))
	if textValue == "" || textValue == "-" {
		return decimal.Zero, nil
	}
	parsed, err := decimal.NewFromString(textValue)
	if err != nil || parsed.IsNegative() {
		return decimal.Zero, fmt.Errorf("invalid non-negative decimal")
	}
	return parsed, nil
}

func classifyStatus(status int) (string, string, bool) {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "UNAUTHORIZED", "Adjust API Token 无效或无权访问该 App", false
	case http.StatusTooManyRequests:
		return "RATE_LIMITED", "Adjust 请求达到限流，请稍后重试", true
	case http.StatusBadRequest:
		return "INVALID_REQUEST", "Adjust 报表参数或事件指标映射无效", false
	default:
		if status >= 500 {
			return "UPSTREAM_UNAVAILABLE", "Adjust 服务暂时不可用", true
		}
		return "UPSTREAM_ERROR", "Adjust 返回异常状态", false
	}
}

func backoff(attempt int) time.Duration { return time.Duration(1<<attempt) * 200 * time.Millisecond }

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
