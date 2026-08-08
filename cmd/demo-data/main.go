package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const gameID = "30000000-0000-4000-8000-000000000001"

type payload struct {
	GameID   string           `json:"game_id"`
	Source   string           `json:"source"`
	FileName string           `json:"file_name"`
	Records  []map[string]any `json:"records"`
}

func main() {
	out := flag.String("out", "examples/generated", "output directory")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		panic(err)
	}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	metaAd, googleAd, tiktokAd := []map[string]any{}, []map[string]any{}, []map[string]any{}
	mmp, revenue, creative := []map[string]any{}, []map[string]any{}, []map[string]any{}
	for day := 0; day < 30; day++ {
		date := start.AddDate(0, 0, day).Format("2006-01-02")
		metaSpend, metaInstalls := 2600+day*55, 410-day*5
		googleSpend, googleInstalls := 1800, 300+day%5
		tiktokSpend, tiktokInstalls := 1350+day%4*30, 230+day%7
		metaAd = append(metaAd, ad(date, "meta-us-001", "US", metaSpend, 100000+day*800, 1600-day*12, metaInstalls))
		googleAd = append(googleAd, ad(date, "google-jp-001", "JP", googleSpend, 120000+day*300, 1800+day%6*10, googleInstalls))
		tiktokAd = append(tiktokAd, ad(date, "tiktok-kr-001", "KR", tiktokSpend, 90000+day*400, 1350+day%5*12, tiktokInstalls))
		mmp = append(mmp,
			mmpRow(date, "meta-us-001", "US", metaInstalls*88/100, 280, 24, metaSpend*105/100),
			mmpRow(date, "google-jp-001", "JP", googleInstalls, 275, 42, googleSpend*150/100),
			mmpRow(date, "tiktok-kr-001", "KR", tiktokInstalls, 205, 29, tiktokSpend*128/100),
		)
		revenue = append(revenue,
			revenueRow(date, "meta-us-001", "US", 800, 560, 24, metaSpend),
			revenueRow(date, "google-jp-001", "JP", 760, 590, 43, googleSpend*150/105),
			revenueRow(date, "tiktok-kr-001", "KR", 620, 470, 30, tiktokSpend*128/105),
		)
		clicks := 920
		if day >= 23 {
			clicks = 920 - (day-22)*100
		}
		creative = append(creative, map[string]any{"date": date, "campaign_external_id": "meta-us-001", "creative_external_id": "creative-video-302", "country": "US", "currency": "USD", "spend": metaSpend * 7 / 10, "impressions": 70000, "clicks": clicks, "installs": metaInstalls * 7 / 10, "frequency": fmt.Sprintf("%.2f", 2.0+float64(day)*2.8/29)})
	}
	files := []payload{
		{gameID, "META", "meta_ad_metrics.json", metaAd}, {gameID, "GOOGLE", "google_ad_metrics.json", googleAd}, {gameID, "TIKTOK", "tiktok_ad_metrics.json", tiktokAd},
		{gameID, "APPSFLYER", "mmp_metrics.json", mmp}, {gameID, "GAME", "game_revenue.json", revenue}, {gameID, "META", "creative_metrics.json", creative},
	}
	for _, file := range files {
		writeJSON(filepath.Join(*out, file.FileName), file)
	}
	writeCSV(filepath.Join(*out, "ad_metrics_sample.csv"), []string{"date", "campaign_external_id", "country", "currency", "spend", "impressions", "clicks", "installs"}, metaAd)
	writeCSV(filepath.Join(*out, "mmp_metrics_sample.csv"), []string{"date", "campaign_external_id", "country", "currency", "installs", "activations", "payers", "revenue"}, mmp[:30])
	writeCSV(filepath.Join(*out, "game_revenue_sample.csv"), []string{"date", "campaign_external_id", "country", "currency", "registrations", "active_users", "payers", "revenue_d1", "revenue_d3", "revenue_d7"}, revenue[:30])
	writeCSV(filepath.Join(*out, "creative_metrics_sample.csv"), []string{"date", "campaign_external_id", "creative_external_id", "country", "currency", "spend", "impressions", "clicks", "installs", "frequency"}, creative)
	fmt.Printf("generated demo files in %s\n", *out)
}

func ad(date, campaign, country string, spend, impressions, clicks, installs int) map[string]any {
	return map[string]any{"date": date, "campaign_external_id": campaign, "country": country, "currency": "USD", "spend": spend, "impressions": impressions, "clicks": clicks, "installs": installs}
}
func mmpRow(date, campaign, country string, installs, activations, payers, revenue int) map[string]any {
	return map[string]any{"date": date, "campaign_external_id": campaign, "country": country, "currency": "USD", "installs": installs, "activations": activations, "payers": payers, "revenue": revenue}
}
func revenueRow(date, campaign, country string, registrations, active, payers, revenueD7 int) map[string]any {
	return map[string]any{"date": date, "campaign_external_id": campaign, "country": country, "currency": "USD", "registrations": registrations, "active_users": active, "payers": payers, "revenue_d1": revenueD7 * 28 / 105, "revenue_d3": revenueD7 * 65 / 105, "revenue_d7": revenueD7}
}
func writeJSON(path string, value payload) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		panic(err)
	}
}
func writeCSV(path string, header []string, rows []map[string]any) {
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write(header); err != nil {
		panic(err)
	}
	for _, row := range rows {
		values := make([]string, len(header))
		for i, key := range header {
			values[i] = stringify(row[key])
		}
		if err := writer.Write(values); err != nil {
			panic(err)
		}
	}
}
func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	default:
		return fmt.Sprint(typed)
	}
}
