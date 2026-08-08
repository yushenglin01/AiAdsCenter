package provider

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Record map[string]string

type DataProvider interface {
	Name() string
	Decode(reader io.Reader) ([]Record, error)
}

type CSVProvider struct{}

func (CSVProvider) Name() string { return "csv" }
func (CSVProvider) Decode(reader io.Reader) ([]Record, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	for i := range header {
		header[i] = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(header[i], "\ufeff")))
	}
	var records []Record
	for line := 2; ; line++ {
		values, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV line %d: %w", line, err)
		}
		if len(values) != len(header) {
			return nil, fmt.Errorf("CSV line %d has %d columns, expected %d", line, len(values), len(header))
		}
		record := make(Record, len(header))
		for i := range header {
			record[header[i]] = strings.TrimSpace(values[i])
		}
		records = append(records, record)
	}
	return records, nil
}

type JSONProvider struct{}

func (JSONProvider) Name() string { return "json" }
func (JSONProvider) Decode(reader io.Reader) ([]Record, error) {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	var rows []map[string]any
	if err := decoder.Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode JSON records: %w", err)
	}
	records := make([]Record, 0, len(rows))
	for index, row := range rows {
		record := make(Record, len(row))
		for key, value := range row {
			normalizedKey := strings.ToLower(strings.TrimSpace(key))
			switch typed := value.(type) {
			case string:
				record[normalizedKey] = strings.TrimSpace(typed)
			case json.Number:
				record[normalizedKey] = typed.String()
			case bool:
				record[normalizedKey] = strconv.FormatBool(typed)
			case nil:
				record[normalizedKey] = ""
			default:
				return nil, fmt.Errorf("JSON row %d field %s must be scalar", index+1, key)
			}
		}
		records = append(records, record)
	}
	return records, nil
}

type MockProvider struct{ Records []Record }

func (m MockProvider) Name() string { return "mock" }
func (m MockProvider) Decode(io.Reader) ([]Record, error) {
	return append([]Record(nil), m.Records...), nil
}

func DecodeBytes(provider DataProvider, data []byte) ([]Record, error) {
	return provider.Decode(bytes.NewReader(data))
}
