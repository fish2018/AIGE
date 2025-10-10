package main

import (
	"AIGE/config"
	"AIGE/models"
	"AIGE/routes"
	"AIGE/utils"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	config.InitDB()

	// 自动迁移
	models.AutoMigrate()

	// 创建默认管理员用户
	utils.CreateDefaultAdmin()

	// 设置路由
	r := gin.Default()

	// 配置CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// 注册路由
	routes.SetupRoutes(r)

	log.Println("服务器启动在端口 :8182")
	r.Run(":8182")
}