package controllers

import (
	"AIGE/config"
	"AIGE/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

type ChatResponse struct {
	ID       uint   `json:"id"`
	Message  string `json:"message"`
	Response string `json:"response"`
}

func SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 这里是AI对话的占位实现，实际使用时可以接入真实的AI API
	aiResponse := "这是一个AI对话系统的演示响应。您说: " + req.Message + "。实际部署时，这里会连接到真实的AI服务。"

	// 保存对话记录
	chatMessage := models.ChatMessage{
		UserID:   userID.(uint),
		Message:  req.Message,
		Response: aiResponse,
	}

	if err := config.DB.Create(&chatMessage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "对话保存失败"})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		ID:       chatMessage.ID,
		Message:  chatMessage.Message,
		Response: chatMessage.Response,
	})
}

func GetChatHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	var messages []models.ChatMessage
	if err := config.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(50).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取对话历史失败"})
		return
	}

	var chatHistory []ChatResponse
	for _, msg := range messages {
		chatHistory = append(chatHistory, ChatResponse{
			ID:       msg.ID,
			Message:  msg.Message,
			Response: msg.Response,
		})
	}

	c.JSON(http.StatusOK, gin.H{"messages": chatHistory})
}