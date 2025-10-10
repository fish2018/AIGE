package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ModelFetcher struct {
	client *http.Client
}

func NewModelFetcher() *ModelFetcher {
	return &ModelFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (mf *ModelFetcher) GetModels(apiType, baseURL, apiKey string) ([]string, error) {
	switch apiType {
	case "openai":
		return mf.getOpenAIModels(baseURL, apiKey)
	case "anthropic":
		return mf.getAnthropicModels(), nil
	case "google":
		return mf.getGoogleModels(baseURL, apiKey)
	default:
		return mf.getOpenAIModels(baseURL, apiKey)
	}
}

func (mf *ModelFetcher) getOpenAIModels(baseURL, apiKey string) ([]string, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("API URL未配置")
	}

	modelsURL := mf.buildOpenAIModelsURL(baseURL)

	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := mf.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI Models API error: %d %s, %s", resp.StatusCode, resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if data, ok := result["data"].([]interface{}); ok {
		models := make([]string, 0)
		for _, item := range data {
			if model, ok := item.(map[string]interface{}); ok {
				if id, ok := model["id"].(string); ok && id != "" {
					models = append(models, id)
				}
			}
		}
		return models, nil
	}

	return nil, fmt.Errorf("OpenAI API返回的模型列表格式不正确")
}

func (mf *ModelFetcher) getGoogleModels(baseURL, apiKey string) ([]string, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("API URL未配置")
	}

	modelsURL := mf.buildGoogleModelsURL(baseURL)
	modelsURL += "?key=" + apiKey

	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := mf.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini Models API error: %d %s, %s", resp.StatusCode, resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if modelsData, ok := result["models"].([]interface{}); ok {
		models := make([]string, 0)
		for _, item := range modelsData {
			if model, ok := item.(map[string]interface{}); ok {
				if name, ok := model["name"].(string); ok {
					name = strings.TrimPrefix(name, "models/")
					if name != "" {
						models = append(models, name)
					}
				}
			}
		}
		return models, nil
	}

	return nil, fmt.Errorf("Gemini API返回的模型列表格式不正确")
}

func (mf *ModelFetcher) getAnthropicModels() []string {
	return []string{
		"claude-opus-4-1-20250805",
		"claude-3-7-sonnet-20250219",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}

func (mf *ModelFetcher) buildOpenAIModelsURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)

	if strings.Contains(baseURL, "/models") {
		return baseURL
	}

	if strings.Contains(baseURL, "/v1") {
		return strings.TrimSuffix(baseURL, "/") + "/models"
	}

	if strings.Contains(baseURL, "/chat/completions") {
		return strings.Replace(baseURL, "/chat/completions", "/models", 1)
	}

	return strings.TrimSuffix(baseURL, "/") + "/v1/models"
}

func (mf *ModelFetcher) buildGoogleModelsURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)

	if strings.Contains(baseURL, "/models") {
		return baseURL
	}

	if strings.Contains(baseURL, "/v1beta") {
		return strings.TrimSuffix(baseURL, "/") + "/models"
	}

	return strings.TrimSuffix(baseURL, "/") + "/v1beta/models"
}
