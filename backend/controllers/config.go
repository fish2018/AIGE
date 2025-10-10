package controllers

import (
	"AIGE/config"
	"AIGE/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSystemConfig 获取系统配置
func GetSystemConfig(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少key参数"})
		return
	}

	var conf models.SystemConfig
	if err := config.DB.Where("key = ?", key).First(&conf).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	c.JSON(http.StatusOK, conf)
}

// GetAllSystemConfigs 获取所有系统配置
func GetAllSystemConfigs(c *gin.Context) {
	var configs []models.SystemConfig
	if err := config.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取配置失败"})
		return
	}

	// 转换为map格式方便前端使用
	configMap := make(map[string]string)
	for _, conf := range configs {
		configMap[conf.Key] = conf.Value
	}

	c.JSON(http.StatusOK, configMap)
}

// SetSystemConfig 设置系统配置
func SetSystemConfig(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 查找或创建配置
	var conf models.SystemConfig
	result := config.DB.Where("key = ?", req.Key).First(&conf)
	
	if result.Error != nil {
		// 不存在，创建新配置
		conf = models.SystemConfig{
			Key:   req.Key,
			Value: req.Value,
		}
		if err := config.DB.Create(&conf).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建配置失败"})
			return
		}
	} else {
		// 存在，更新配置
		conf.Value = req.Value
		if err := config.DB.Save(&conf).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新配置失败"})
			return
		}
	}

	c.JSON(http.StatusOK, conf)
}

// BatchSetSystemConfig 批量设置系统配置
func BatchSetSystemConfig(c *gin.Context) {
	var req map[string]string

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 批量更新配置
	for key, value := range req {
		var conf models.SystemConfig
		result := config.DB.Where("key = ?", key).First(&conf)
		
		if result.Error != nil {
			// 不存在，创建新配置
			conf = models.SystemConfig{
				Key:   key,
				Value: value,
			}
			config.DB.Create(&conf)
		} else {
			// 存在，更新配置
			conf.Value = value
			config.DB.Save(&conf)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置更新成功"})
}
