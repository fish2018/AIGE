package routes

import (
	"AIGE/controllers"
	"AIGE/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// 公开路由
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
	}

	// 需要认证的路由
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// 用户相关
		api.GET("/profile", controllers.GetProfile)
		
		// AI对话相关
		api.POST("/chat", controllers.SendMessage)
		api.GET("/chat/history", controllers.GetChatHistory)

		// 游戏相关
		api.GET("/game/mods", controllers.GetAvailableMods)
		api.POST("/game/init", controllers.InitializeGame)
		api.GET("/game/ws", controllers.GameWebSocket)
		api.GET("/game/state", controllers.GetGameState)
		api.DELETE("/game/reset", controllers.ResetGame)
		api.POST("/game/save", controllers.ManualSaveGame)
		api.POST("/game/restart-opportunities", controllers.RestartOpportunities)
	}

	// 管理员路由
	admin := api.Group("/admin")
	admin.Use(middleware.AdminMiddleware())
	{
		admin.GET("/users", controllers.GetUsers)
		admin.GET("/users/:id", controllers.GetUser)
		admin.PUT("/users/:id/password", controllers.UpdateUserPassword)
		admin.DELETE("/users/:id", controllers.DeleteUser)
		admin.PUT("/users/:id/toggle-admin", controllers.ToggleUserAdmin)

		admin.GET("/providers", controllers.GetProviders)
		admin.GET("/providers/:id", controllers.GetProvider)
		admin.POST("/providers", controllers.CreateProvider)
		admin.PUT("/providers/:id", controllers.UpdateProvider)
		admin.DELETE("/providers/:id", controllers.DeleteProvider)
		admin.PUT("/providers/:id/toggle", controllers.ToggleProvider)
		admin.GET("/providers/:id/models/available", controllers.GetAvailableModels)
		admin.GET("/providers/:id/test", controllers.TestConnection)

		admin.GET("/models", controllers.GetModels)
		admin.GET("/models/:id", controllers.GetModel)
		admin.POST("/models", controllers.CreateModel)
		admin.PUT("/models/:id", controllers.UpdateModel)
		admin.DELETE("/models/:id", controllers.DeleteModel)
		admin.PUT("/models/:id/toggle", controllers.ToggleModel)
		admin.POST("/models/:id/test", controllers.TestModel)
		admin.PUT("/models/:id/capabilities", controllers.UpdateModelCapabilities)

		admin.POST("/ai/chat", controllers.ChatWithAI)
		admin.POST("/ai/test", controllers.TestModelConnection)

		// 系统配置
		admin.GET("/config", controllers.GetAllSystemConfigs)
		admin.GET("/config/:key", controllers.GetSystemConfig)
		admin.POST("/config", controllers.SetSystemConfig)
		admin.POST("/config/batch", controllers.BatchSetSystemConfig)
		
		// 游戏配置管理
		admin.POST("/game/reload-config", controllers.ReloadGameConfig)
		admin.GET("/game/model-config", controllers.GetGameModelConfig)
		admin.POST("/game/model-config", controllers.SaveGameModelConfig)
	}
}