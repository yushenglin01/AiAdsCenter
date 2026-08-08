package handler

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/ingestion/contract"
	"github.com/example/adnova/internal/ingestion/dto"
	"github.com/example/adnova/internal/ingestion/service"
	"github.com/gin-gonic/gin"
)

const maxUploadBytes = 10 << 20

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) ImportAd(c *gin.Context)       { h.importData(c, service.ImportAd) }
func (h *Handler) ImportMMP(c *gin.Context)      { h.importData(c, service.ImportMMP) }
func (h *Handler) ImportRevenue(c *gin.Context)  { h.importData(c, service.ImportRevenue) }
func (h *Handler) ImportCreative(c *gin.Context) { h.importData(c, service.ImportCreative) }
func (h *Handler) ImportBatch(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Fail(c, apperror.Validation("标准批次 JSON 不能超过 10MB"))
		return
	}
	batch, err := contract.Decode(payload)
	if err != nil {
		response.Fail(c, apperror.Validation("标准批次 JSON 无效"))
		return
	}
	job, err := h.service.ImportBatch(c, identity.TenantID(c), identity.UserID(c), batch)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, job)
}
func (h *Handler) List(c *gin.Context) {
	rows, err := h.service.List(c, identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}
func (h *Handler) Get(c *gin.Context) {
	row, err := h.service.Get(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}

func (h *Handler) importData(c *gin.Context, importType string) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)
	input, err := readInput(c, importType)
	if err != nil {
		response.Fail(c, apperror.Validation(err.Error()))
		return
	}
	input.TenantID, input.UserID = identity.TenantID(c), identity.UserID(c)
	job, err := h.service.Import(c, input)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, job)
}

func readInput(c *gin.Context, importType string) (service.ImportInput, error) {
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return service.ImportInput{}, fmt.Errorf("缺少 file: %w", err)
		}
		if fileHeader.Size > maxUploadBytes {
			return service.ImportInput{}, fmt.Errorf("文件不能超过 10MB")
		}
		data, err := readMultipart(fileHeader)
		if err != nil {
			return service.ImportInput{}, err
		}
		return service.ImportInput{GameID: c.PostForm("game_id"), Source: c.PostForm("source"), FileName: filepath.Base(fileHeader.Filename), Data: data, ImportType: importType}, nil
	}
	var req dto.JSONImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.ImportInput{}, fmt.Errorf("JSON 请求必须包含 game_id、source、file_name 和 records")
	}
	if len(req.Records) > maxUploadBytes {
		return service.ImportInput{}, fmt.Errorf("JSON 数据不能超过 10MB")
	}
	return service.ImportInput{GameID: req.GameID, Source: req.Source, FileName: filepath.Base(req.FileName), Data: req.Records, ImportType: importType}, nil
}

func readMultipart(header *multipart.FileHeader) ([]byte, error) {
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxUploadBytes {
		return nil, fmt.Errorf("文件不能超过 10MB")
	}
	return data, nil
}
