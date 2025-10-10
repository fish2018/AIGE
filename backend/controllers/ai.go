package controllers

import (
	"AIGE/config"
	"AIGE/models"
	"AIGE/services"
	"bufio"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ChatWithAI(c *gin.Context) {
	var request struct {
		ProviderID uint                    `json:"provider_id"`
		ModelID    string                  `json:"model_id"`
		Messages   []services.Message      `json:"messages"`
		Stream     bool                    `json:"stream"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var provider models.Provider
	if err := config.DB.First(&provider, request.ProviderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if provider.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider API key not configured"})
		return
	}

	var model models.Model
	if err := config.DB.Where("provider_id = ? AND model_id = ?", request.ProviderID, request.ModelID).First(&model).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	aiClient := services.NewAIClient()

	apiType := model.APIType
	if apiType == "" {
		apiType = provider.Type
	}

	baseURL := provider.BaseURL
	if baseURL == "" {
		switch apiType {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "anthropic":
			baseURL = "https://api.anthropic.com/v1"
		case "google":
			baseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
	}

	var result interface{}
	var err error

	switch apiType {
	case "openai":
		result, err = aiClient.CallOpenAI(baseURL, provider.APIKey, model.ModelID, request.Messages, request.Stream)
	case "anthropic":
		result, err = aiClient.CallAnthropic(baseURL, provider.APIKey, model.ModelID, request.Messages, request.Stream)
	case "google":
		result, err = aiClient.CallGoogle(baseURL, provider.APIKey, model.ModelID, request.Messages, request.Stream)
	default:
		result, err = aiClient.CallOpenAI(baseURL, provider.APIKey, model.ModelID, request.Messages, request.Stream)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if request.Stream {
		body, ok := result.(io.ReadCloser)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid stream response"})
			return
		}
		defer body.Close()

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		scanner := bufio.NewScanner(body)
		buf := make([]byte, 0, 128*1024)
		scanner.Buffer(buf, 2*1024*1024) // 2MB最大token大小
		
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			chunk := aiClient.ParseStreamChunk(apiType, line)
			if chunk != nil {
				c.SSEvent("message", chunk)
				c.Writer.Flush()

				if done, ok := chunk["done"].(bool); ok && done {
					break
				}
			}
		}
		
		if err := scanner.Err(); err != nil {
			c.SSEvent("error", gin.H{"error": err.Error()})
			c.Writer.Flush()
		}
	} else {
		c.JSON(http.StatusOK, result)
	}
}

func TestModelConnection(c *gin.Context) {
	var request struct {
		ProviderID uint   `json:"provider_id"`
		ModelID    string `json:"model_id"`
		APIType    string `json:"api_type"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var provider models.Provider
	if err := config.DB.First(&provider, request.ProviderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if provider.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider API key not configured"})
		return
	}

	apiType := request.APIType
	if apiType == "" {
		apiType = provider.Type
	}

	baseURL := provider.BaseURL
	if baseURL == "" {
		switch apiType {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "anthropic":
			baseURL = "https://api.anthropic.com/v1"
		case "google":
			baseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
	}

	testMessages := []services.Message{
		{
			Role:    "user",
			Content: "Hello, this is a connection test. Please reply with 'OK'.",
		},
	}

	aiClient := services.NewAIClient()

	var result interface{}
	var err error

	switch apiType {
	case "openai":
		result, err = aiClient.CallOpenAI(baseURL, provider.APIKey, request.ModelID, testMessages, false)
	case "anthropic":
		result, err = aiClient.CallAnthropic(baseURL, provider.APIKey, request.ModelID, testMessages, false)
	case "google":
		result, err = aiClient.CallGoogle(baseURL, provider.APIKey, request.ModelID, testMessages, false)
	default:
		result, err = aiClient.CallOpenAI(baseURL, provider.APIKey, request.ModelID, testMessages, false)
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection test successful",
		"result":  result,
	})
}
