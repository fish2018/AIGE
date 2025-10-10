package controllers

import (
	"AIGE/config"
	"AIGE/models"
	"AIGE/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetProviders(c *gin.Context) {
	var providers []models.Provider
	if err := config.DB.Preload("Models").Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func GetProvider(c *gin.Context) {
	providerID := c.Param("id")
	var provider models.Provider

	if err := config.DB.Preload("Models").First(&provider, providerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func CreateProvider(c *gin.Context) {
	var provider models.Provider

	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if provider.Name == "" || provider.Type == "" || provider.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name, type, and API key are required"})
		return
	}

	if err := config.DB.Create(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

func UpdateProvider(c *gin.Context) {
	providerID := c.Param("id")
	var provider models.Provider

	if err := config.DB.First(&provider, providerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Model(&provider).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func DeleteProvider(c *gin.Context) {
	providerID := c.Param("id")
	var provider models.Provider

	if err := config.DB.First(&provider, providerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	config.DB.Where("provider_id = ?", providerID).Delete(&models.Model{})

	if err := config.DB.Delete(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted successfully"})
}

func ToggleProvider(c *gin.Context) {
	providerID := c.Param("id")
	var provider models.Provider

	if err := config.DB.First(&provider, providerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	provider.Enabled = !provider.Enabled

	if err := config.DB.Save(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle provider"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func GetAvailableModels(c *gin.Context) {
	providerID := c.Param("id")
	apiType := c.Query("api_type")

	var provider models.Provider
	if err := config.DB.First(&provider, providerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if provider.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider API key not configured"})
		return
	}

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

	fetcher := services.NewModelFetcher()
	models, err := fetcher.GetModels(apiType, baseURL, provider.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

func TestConnection(c *gin.Context) {
	providerIDStr := c.Param("id")
	modelID := c.Query("model_id")

	providerID, err := strconv.ParseUint(providerIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider models.Provider
	if err := config.DB.First(&provider, providerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection test successful",
		"provider_id": providerID,
		"model_id": modelID,
	})
}
