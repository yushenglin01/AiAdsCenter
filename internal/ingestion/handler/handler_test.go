package handler

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/example/adnova/internal/ingestion/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestReadJSONInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"game_id":"game-1","source":"META","file_name":"../safe.json","records":[{"date":"2026-07-01"}]}`
	request := httptest.NewRequest("POST", "/imports/ad-metrics", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	input, err := readInput(context, service.ImportAd)
	require.NoError(t, err)
	require.Equal(t, "safe.json", input.FileName, "path components must be removed")
	require.Equal(t, service.ImportAd, input.ImportType)
}

func TestReadJSONInputRequiresRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("POST", "/imports/ad-metrics", bytes.NewBufferString(`{"game_id":"game-1","source":"META","file_name":"safe.json"}`))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	_, err := readInput(context, service.ImportAd)
	require.Error(t, err)
}
