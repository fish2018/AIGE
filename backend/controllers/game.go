package controllers

import (
	"AIGE/config"
	"AIGE/game_engine"
	"AIGE/models"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	modLoader    *game_engine.ModLoader
	stateManager *game_engine.StateManager
	gameController *game_engine.GameController
	initOnce     sync.Once
)

// 初始化游戏引擎
func InitGameEngine() {
	initOnce.Do(func() {
		// 初始化Mod加载器（mods目录在项目根目录，backend在backend子目录）
		modsPath := "../mods"
		modLoader = game_engine.NewModLoader(modsPath)
		if err := modLoader.LoadMods(modsPath); err != nil {
			fmt.Printf("加载mods失败: %v\n", err)
		} else {
			fmt.Printf("游戏引擎初始化完成，已加载 %d 个mod\n", len(modLoader.GetAllMods()))
		}

		// 初始化状态管理器
		stateManager = game_engine.NewStateManager(true, 5*time.Minute)

		// 初始化游戏控制器
		gameController = game_engine.NewGameController(modLoader, stateManager)
		
		// GameController 在初始化时会自动加载所有模型配置到内存
	})
}

// 注释：已移除旧的 configureAIProvider 函数
// 现在使用 GameController 的内存缓存机制动态管理模型配置

// GetAvailableMods 获取可用的游戏mod列表
func GetAvailableMods(c *gin.Context) {
	InitGameEngine()

	mods := modLoader.GetAllMods()
	modList := make([]map[string]interface{}, 0)

	for _, mod := range mods {
		modInfo := map[string]interface{}{
			"game_id":     mod.Config.GameID,
			"name":        mod.Config.Name,
			"version":     mod.Config.Version,
			"description": mod.Config.Description,
			"author":      mod.Config.Author,
		}
		modList = append(modList, modInfo)
	}

	c.JSON(http.StatusOK, modList)
}

// InitializeGame 初始化游戏会话
func InitializeGame(c *gin.Context) {
	InitGameEngine()

	// 从请求中获取用户ID和modID
	var req struct {
		ModID string `json:"mod_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 从token中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 初始化或获取游戏会话
	session, err := gameController.InitializeGame(fmt.Sprintf("%v", userID), req.ModID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"state": session,
	})
}

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应该检查origin
	},
}

// GameWebSocket WebSocket连接处理
func GameWebSocket(c *gin.Context) {
	InitGameEngine()

	// 获取用户ID和modID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	modID := c.Query("mod_id")
	if modID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少mod_id参数"})
		return
	}

	// 升级为WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("WebSocket升级失败: %v\n", err)
		return
	}
	defer conn.Close()

	playerID := fmt.Sprintf("%v", userID)
	fmt.Printf("玩家 %s 连接到 mod %s\n", playerID, modID)

	// 发送当前状态
	session, err := stateManager.GetSession(playerID, modID)
	if err == nil {
		sendMessage(conn, "full_state", session)
	}

	// 处理WebSocket消息
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket错误: %v\n", err)
			}
			break
		}

		action, ok := message["action"].(string)
		if !ok {
			sendError(conn, "无效的消息格式")
			continue
		}

		// 流式回调函数
		streamCallback := func(chunk string) error {
			// 检查是否是判定结果
			if strings.HasPrefix(chunk, "【判定结果：") {
				return sendMessage(conn, "roll_result", map[string]interface{}{
					"content": chunk,
				})
			}
			return sendMessage(conn, "narrative_chunk", map[string]interface{}{
				"content": chunk,
			})
		}
		
		// 第二阶段叙事回调函数（作为新消息）
		secondStageCallback := func(chunk string) error {
			return sendMessage(conn, "second_stage_narrative", map[string]interface{}{
				"content": chunk,
			})
		}

		// 判定事件回调函数
		rollCallback := func(rollEvent map[string]interface{}) error {
			return sendMessage(conn, "roll_event", rollEvent)
		}

		// 处理不同的动作 - 统一使用流式处理
		err = gameController.ProcessActionStream(playerID, modID, action, streamCallback, rollCallback, secondStageCallback)

		if err != nil {
			sendError(conn, err.Error())
			continue
		}

		// 发送更新后的状态
		session, err := stateManager.GetSession(playerID, modID)
		if err != nil {
			sendError(conn, "获取会话状态失败")
			continue
		}

		sendMessage(conn, "full_state", session)
	}

	fmt.Printf("玩家 %s 断开连接\n", playerID)
}

// sendMessage 发送WebSocket消息
func sendMessage(conn *websocket.Conn, msgType string, data interface{}) error {
	message := map[string]interface{}{
		"type": msgType,
		"data": data,
	}
	if err := conn.WriteJSON(message); err != nil {
		fmt.Printf("发送消息失败: %v\n", err)
		return err
	}
	return nil
}

// sendError 发送错误消息
func sendError(conn *websocket.Conn, detail string) {
	message := map[string]interface{}{
		"type":   "error",
		"detail": detail,
	}
	if err := conn.WriteJSON(message); err != nil {
		fmt.Printf("发送错误消息失败: %v\n", err)
	}
}

// GetGameState 获取游戏状态（用于调试）
func GetGameState(c *gin.Context) {
	InitGameEngine()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	modID := c.Query("mod_id")
	if modID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少mod_id参数"})
		return
	}

	playerID := fmt.Sprintf("%v", userID)
	session, err := stateManager.GetSession(playerID, modID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// ResetGame 重置游戏（用于测试）
func ResetGame(c *gin.Context) {
	InitGameEngine()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	modID := c.Query("mod_id")
	if modID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少mod_id参数"})
		return
	}

	playerID := fmt.Sprintf("%v", userID)
	
	if err := stateManager.DeleteSession(playerID, modID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除会话失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "游戏已重置"})
}

// ManualSaveGame 手动保存游戏
func ManualSaveGame(c *gin.Context) {
	InitGameEngine()
	
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	
	var req struct {
		ModID string `json:"mod_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	
	// 保存当前会话到文件
	if err := stateManager.SaveToFile(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "保存成功",
		"saved_at": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ReloadGameConfig 重新加载游戏AI配置（管理员接口）
func ReloadGameConfig(c *gin.Context) {
	fmt.Println("[ReloadGameConfig] 开始重新加载游戏AI配置...")
	
	// 确保游戏引擎已初始化
	InitGameEngine()
	
	// 重新加载所有游戏模型配置到内存
	if gameController != nil {
		gameController.LoadAllGameModelConfigs()
		fmt.Printf("✅ 游戏AI配置已重新加载并生效\n")
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "游戏AI配置已重新加载并生效"})
}

// GetGameModelConfig 获取游戏AI模型配置（管理员接口）
func GetGameModelConfig(c *gin.Context) {
	db := config.DB
	
	// 获取默认模型ID配置
	var gameModelConfig models.SystemConfig
	var defaultModelID string
	
	err := db.Where("key = ?", "game_model_id").First(&gameModelConfig).Error
	if err == nil && gameModelConfig.Value != "" {
		// 验证配置的模型是否仍然存在且启用
		var model models.Model
		if err := db.Where("id = ? AND enabled = ?", gameModelConfig.Value, true).First(&model).Error; err == nil {
			defaultModelID = gameModelConfig.Value
			fmt.Printf("[GetGameModelConfig] 从数据库读取到游戏模型配置：model_id = %s\n", defaultModelID)
		} else {
			fmt.Printf("[GetGameModelConfig] 配置的模型(ID: %s)不存在或未启用，使用默认模型\n", gameModelConfig.Value)
		}
	}
	
	// 如果没有有效的已保存配置，查找第一个启用的模型作为默认模型
	if defaultModelID == "" {
		var defaultModel models.Model
		if err := db.Where("enabled = ?", true).First(&defaultModel).Error; err == nil {
			defaultModelID = fmt.Sprintf("%d", defaultModel.ID)
			fmt.Printf("[GetGameModelConfig] 使用第一个启用的模型作为默认：model_id = %s\n", defaultModelID)
		} else {
			fmt.Printf("[GetGameModelConfig] 警告：没有找到任何启用的模型\n")
		}
	}
	
	// 获取游戏专用模型配置
	gameModels := make(map[string]string)
	var gameSpecificConfigs []models.SystemConfig
	err = db.Where("key LIKE ?", "game_model_%").Find(&gameSpecificConfigs).Error
	if err == nil {
		for _, config := range gameSpecificConfigs {
			if config.Key != "game_model_id" && config.Value != "" {
				// 从 game_model_xiuxian2 提取 xiuxian2
				if len(config.Key) > 11 { // "game_model_" 的长度是11
					modID := config.Key[11:]
					// 验证模型是否存在且启用
					var model models.Model
					if err := db.Where("id = ? AND enabled = ?", config.Value, true).First(&model).Error; err == nil {
						gameModels[modID] = config.Value
						fmt.Printf("[GetGameModelConfig] 加载游戏专用模型配置：%s = %s\n", modID, config.Value)
					} else {
						fmt.Printf("[GetGameModelConfig] 游戏 %s 配置的模型(ID: %s)不存在或未启用\n", modID, config.Value)
					}
				}
			}
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"default_model_id": defaultModelID,
		"game_models": gameModels,
	})
}

// SaveGameModelConfig 保存游戏AI模型配置（管理员接口）
func SaveGameModelConfig(c *gin.Context) {
	var req struct {
		DefaultModelID string            `json:"default_model_id"`
		GameModels     map[string]string `json:"game_models"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	db := config.DB
	
	// 验证默认模型ID是否存在
	if req.DefaultModelID != "" {
		var model models.Model
		if err := db.First(&model, req.DefaultModelID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "指定的默认模型不存在"})
			return
		}
		
		// 保存默认模型配置
		var gameModelConfig models.SystemConfig
		err := db.Where("key = ?", "game_model_id").First(&gameModelConfig).Error
		if err != nil {
			// 如果不存在，创建新记录
			gameModelConfig = models.SystemConfig{
				Key:   "game_model_id",
				Value: req.DefaultModelID,
			}
			if err := db.Create(&gameModelConfig).Error; err != nil {
				fmt.Printf("❌ 创建游戏模型配置失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败"})
				return
			}
			fmt.Printf("✅ 创建游戏模型配置：model_id = %s\n", req.DefaultModelID)
		} else {
			// 如果存在，更新记录
			gameModelConfig.Value = req.DefaultModelID
			if err := db.Save(&gameModelConfig).Error; err != nil {
				fmt.Printf("❌ 更新游戏模型配置失败: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败"})
				return
			}
			fmt.Printf("✅ 更新游戏模型配置：model_id = %s\n", req.DefaultModelID)
		}
	}
	
	// 处理游戏专用模型配置
	for modID, modelID := range req.GameModels {
		if modelID != "" {
			// 验证模型是否存在
			var model models.Model
			if err := db.First(&model, modelID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("游戏 %s 指定的模型不存在", modID)})
				return
			}
			
			// 保存游戏专用模型配置
			configKey := fmt.Sprintf("game_model_%s", modID)
			var gameSpecificConfig models.SystemConfig
			err := db.Where("key = ?", configKey).First(&gameSpecificConfig).Error
			if err != nil {
				// 如果不存在，创建新记录
				gameSpecificConfig = models.SystemConfig{
					Key:   configKey,
					Value: modelID,
				}
				if err := db.Create(&gameSpecificConfig).Error; err != nil {
					fmt.Printf("❌ 创建游戏 %s 模型配置失败: %v\n", modID, err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "保存游戏专用配置失败"})
					return
				}
				fmt.Printf("✅ 创建游戏 %s 模型配置：model_id = %s\n", modID, modelID)
			} else {
				// 如果存在，更新记录
				gameSpecificConfig.Value = modelID
				if err := db.Save(&gameSpecificConfig).Error; err != nil {
					fmt.Printf("❌ 更新游戏 %s 模型配置失败: %v\n", modID, err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "保存游戏专用配置失败"})
					return
				}
				fmt.Printf("✅ 更新游戏 %s 模型配置：model_id = %s\n", modID, modelID)
			}
		} else {
			// 如果模型ID为空，删除该游戏的专用配置
			configKey := fmt.Sprintf("game_model_%s", modID)
			if err := db.Where("key = ?", configKey).Delete(&models.SystemConfig{}).Error; err != nil {
				fmt.Printf("❌ 删除游戏 %s 模型配置失败: %v\n", modID, err)
			} else {
				fmt.Printf("✅ 删除游戏 %s 模型配置\n", modID)
			}
		}
	}
	
	// 更新GameController内存缓存
	if gameController != nil {
		// 更新默认模型配置
		if req.DefaultModelID != "" {
			gameController.UpdateDefaultModelConfig(req.DefaultModelID)
		}
		
		// 更新游戏专用模型配置
		for modID, modelID := range req.GameModels {
			gameController.UpdateGameModelConfig(modID, modelID)
		}
		
		fmt.Printf("✅ 游戏AI内存配置已更新\n")
	} else {
		fmt.Printf("⚠️ GameController未初始化，跳过内存配置更新\n")
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已保存"})
}

// RestartOpportunities 重启机缘（清空指定MOD存档，重置机缘次数）
func RestartOpportunities(c *gin.Context) {
	userID := c.GetUint("user_id") // 修复：使用正确的键名
	fmt.Printf("[RestartOpportunities] 获取到的用户ID: %d\n", userID)
	
	if userID == 0 {
		fmt.Printf("[RestartOpportunities] 用户ID为0，认证失败\n")
		// 输出更多调试信息
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			headerLen := len(authHeader)
			if headerLen > 20 {
				fmt.Printf("[RestartOpportunities] Authorization header: %s...\n", authHeader[:20])
			} else {
				fmt.Printf("[RestartOpportunities] Authorization header: %s\n", authHeader)
			}
		} else {
			fmt.Printf("[RestartOpportunities] 没有Authorization header\n")
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 从请求体中获取mod_id
	var req struct {
		ModID string `json:"mod_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少mod_id参数"})
		return
	}

	fmt.Printf("[RestartOpportunities] 用户 %d 请求重启机缘，MOD: %s\n", userID, req.ModID)

	// 只删除该用户在指定MOD的游戏存档
	db := config.DB
	result := db.Unscoped().Where("user_id = ? AND mod_id = ?", userID, req.ModID).Delete(&models.GameSave{})
	if result.Error != nil {
		fmt.Printf("❌ 删除游戏存档失败: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除存档失败"})
		return
	}

	deletedCount := result.RowsAffected
	fmt.Printf("✅ 成功删除用户 %d 在MOD %s 的 %d 条游戏存档\n", userID, req.ModID, deletedCount)

	// 重置游戏引擎中的会话状态（如果存在）
	if gameController != nil && stateManager != nil {
		// 只清除指定MOD的内存会话数据
		playerIDStr := fmt.Sprintf("%d", userID)
		err := stateManager.DeleteSession(playerIDStr, req.ModID)
		if err != nil {
			fmt.Printf("⚠️ 清除MOD %s 内存会话数据失败: %v\n", req.ModID, err)
		} else {
			fmt.Printf("✅ 清除用户 %d 在MOD %s 的内存会话数据\n", userID, req.ModID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("机缘已重启，%s 的存档已清空", req.ModID),
		"deleted_saves": deletedCount,
		"mod_id": req.ModID,
	})
}
