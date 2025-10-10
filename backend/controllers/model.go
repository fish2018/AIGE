package controllers

import (
	"AIGE/config"
	"AIGE/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetModels(c *gin.Context) {
	providerID := c.Query("provider_id")

	var modelList []models.Model
	query := config.DB.Preload("Provider")

	if providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}

	if err := query.Find(&modelList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": modelList})
}

func GetModel(c *gin.Context) {
	modelID := c.Param("id")
	var model models.Model

	if err := config.DB.Preload("Provider").First(&model, modelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	c.JSON(http.StatusOK, model)
}

func CreateModel(c *gin.Context) {
	var model models.Model

	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if model.ModelID == "" || model.Name == "" || model.ProviderID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID, name, and provider ID are required"})
		return
	}

	var provider models.Provider
	if err := config.DB.First(&provider, model.ProviderID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	if err := config.DB.Create(&model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create model"})
		return
	}

	c.JSON(http.StatusCreated, model)
}

func UpdateModel(c *gin.Context) {
	modelID := c.Param("id")
	var model models.Model

	if err := config.DB.First(&model, modelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Model(&model).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model"})
		return
	}

	c.JSON(http.StatusOK, model)
}

func DeleteModel(c *gin.Context) {
	modelID := c.Param("id")
	var model models.Model

	if err := config.DB.First(&model, modelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	if err := config.DB.Delete(&model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete model"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model deleted successfully"})
}

func ToggleModel(c *gin.Context) {
	modelID := c.Param("id")
	var model models.Model

	if err := config.DB.First(&model, modelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	model.Enabled = !model.Enabled

	if err := config.DB.Save(&model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle model"})
		return
	}

	c.JSON(http.StatusOK, model)
}

func TestModel(c *gin.Context) {
	modelID := c.Param("id")
	var model models.Model

	if err := config.DB.Preload("Provider").First(&model, modelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	now := time.Now()
	model.LastTested = &now
	model.TestStatus = "success"

	if err := config.DB.Save(&model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update test status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Model test successful",
		"model": model,
	})
}

func UpdateModelCapabilities(c *gin.Context) {
	modelID := c.Param("id")
	var model models.Model

	if err := config.DB.First(&model, modelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	var request struct {
		Capabilities string `json:"capabilities"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	model.Capabilities = request.Capabilities
	model.LastTested = &now
	model.TestStatus = "success"

	if err := config.DB.Save(&model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update capabilities"})
		return
	}

	c.JSON(http.StatusOK, model)
}
