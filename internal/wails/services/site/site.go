package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"jarvist/internal/common/config"
	"jarvist/pkg/logger"
	"net/http"

	"gorm.io/gorm"
)

type SiteCategoriesResponse struct {
	Success bool           `json:"success"`
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    []SiteCategory `json:"data"`
}

type SiteCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type SiteService struct {
	db     *gorm.DB
	config *config.Config
	logger *logger.ContextLogger
}

func New(db *gorm.DB, cfg *config.Config, logger *logger.ContextLogger) *SiteService {
	return &SiteService{
		db:     db,
		config: cfg,
		logger: logger,
	}
}

func (s *SiteService) ValidateSiteCode(placeCode string) (interface{}, error) {
	requestData := map[string]interface{}{
		"place_code": placeCode,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.ApiUrl+"/v1/app/site-validate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-API-Key", s.config.ApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if code, ok := response["code"].(float64); ok && code == 409 {
		type ErrorResponse struct {
			Success bool                   `json:"success"`
			Code    int                    `json:"code"`
			Message string                 `json:"message"`
			Details map[string]interface{} `json:"details"`
		}

		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error response: %w", err)
		}

		return map[string]interface{}{
			"success": false,
			"message": errorResp.Message,
			"data":    errorResp.Details,
		}, nil
	}

	if success, ok := response["success"].(bool); ok && success {
		return map[string]interface{}{
			"success": true,
			"message": response["message"],
		}, nil
	}

	return nil, fmt.Errorf("unexpected response: %s", string(body))
}

func (s *SiteService) GetSiteCetegories() ([]SiteCategory, error) {
	req, err := http.NewRequest("GET", s.config.ApiUrl+"/v1/app/site-category", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("X-API-Key", s.config.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var response SiteCategoriesResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Periksa apakah request berhasil
	if !response.Success {
		return nil, fmt.Errorf("API returned error: %s", response.Message)
	}

	return response.Data, nil
}
