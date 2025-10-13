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
		
		// 配置AI provider（从数据库获取第一个启用的模型）
		configureAIProvider()
	})
}

// 配置AI提供商
func configureAIProvider() {
	db := config.DB
	
	// 先尝试从系统配置中获取游戏使用的模型ID
	var gameModelConfig models.SystemConfig
	var model models.Model
	
	// 使用 game_model_id 作为配置key（与前端保持一致）
	err := db.Where("key = ?", "game_model_id").First(&gameModelConfig).Error
	if err == nil && gameModelConfig.Value != "" {
		// 使用配置的模型ID
		var modelID uint
		fmt.Sscanf(gameModelConfig.Value, "%d", &modelID)
		
		fmt.Printf("[configureAIProvider] 从系统配置读取到游戏模型ID: %d\n", modelID)
		
		if err := db.Preload("Provider").Where("id = ? AND enabled = ?", modelID, true).First(&model).Error; err != nil {
			fmt.Printf("⚠️  配置的游戏模型(ID: %d)不存在或未启用，尝试使用默认模型\n", modelID)
		} else {
			fmt.Printf("✅ 使用系统配置的游戏模型：%s / %s (ID: %d)\n", model.Provider.Name, model.ModelID, modelID)
			goto SetProvider
		}
	} else {
		fmt.Println("[configureAIProvider] 未找到 game_model_id 配置，将使用第一个启用的模型")
	}
	
	// 如果没有配置或配置的模型不可用，使用第一个启用的模型
	if err := db.Preload("Provider").Where("enabled = ?", true).First(&model).Error; err != nil {
		fmt.Printf("⚠️  未找到启用的AI模型，游戏功能将不可用。请在管理后台配置Provider和Model。\n")
		return
	}
	fmt.Printf("使用默认启用的模型：%s / %s\n", model.Provider.Name, model.ModelID)
	
SetProvider:
	if model.Provider.ID == 0 {
		fmt.Printf("警告：模型 %s 没有关联的Provider\n", model.ModelID)
		return
	}
	
	if !model.Provider.Enabled {
		fmt.Printf("警告：模型 %s 的Provider %s 未启用\n", model.ModelID, model.Provider.Name)
		return
	}
	
	// 优先使用Model的APIType，如果为空则使用Provider的Type
	apiType := model.APIType
	if apiType == "" {
		apiType = model.Provider.Type
	}
	
	// 设置AI provider配置
	provider := game_engine.AIProvider{
		APIType: apiType,
		BaseURL: model.Provider.BaseURL,
		APIKey:  model.Provider.APIKey,
		ModelID: model.ModelID,
	}
	
	gameController.SetAIProvider(provider)
	fmt.Printf("✅ 游戏AI配置成功：%s / %s\n", model.Provider.Name, model.ModelID)
}

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
	
	// 重新配置AI provider
	configureAIProvider()
	
	c.JSON(http.StatusOK, gin.H{"message": "游戏AI配置已重新加载并生效"})
}

// GetGameModelConfig 获取游戏AI模型配置（管理员接口）
func GetGameModelConfig(c *gin.Context) {
	db := config.DB
	
	// 先尝试从数据库获取已保存的配置
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
	
	c.JSON(http.StatusOK, gin.H{
		"default_model_id": defaultModelID,
		"game_models": map[string]interface{}{},
	})
}

// SaveGameModelConfig 保存游戏AI模型配置（管理员接口）
func SaveGameModelConfig(c *gin.Context) {
	var req struct {
		DefaultModelID string                 `json:"default_model_id"`
		GameModels     map[string]interface{} `json:"game_models"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	db := config.DB
	
	// 验证模型ID是否存在
	if req.DefaultModelID != "" {
		var model models.Model
		if err := db.First(&model, req.DefaultModelID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "指定的模型不存在"})
			return
		}
		
		// 保存配置到数据库
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
		
		// 重新配置游戏引擎的AI Provider
		if err := db.Preload("Provider").First(&model, req.DefaultModelID).Error; err == nil {
			apiType := model.APIType
			if apiType == "" {
				apiType = model.Provider.Type
			}
			
			provider := game_engine.AIProvider{
				APIType: apiType,
				BaseURL: model.Provider.BaseURL,
				APIKey:  model.Provider.APIKey,
				ModelID: model.ModelID,
			}
			
			if gameController != nil {
				gameController.SetAIProvider(provider)
				fmt.Printf("✅ 游戏AI配置已更新：%s / %s\n", model.Provider.Name, model.ModelID)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已保存"})
}

// RestartOpportunities 重启机缘（清空存档，重置机缘次数）
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

	fmt.Printf("[RestartOpportunities] 用户 %d 请求重启机缘\n", userID)

	// 删除该用户的所有游戏存档
	db := config.DB
	result := db.Unscoped().Where("user_id = ?", userID).Delete(&models.GameSave{})
	if result.Error != nil {
		fmt.Printf("❌ 删除游戏存档失败: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除存档失败"})
		return
	}

	deletedCount := result.RowsAffected
	fmt.Printf("✅ 成功删除用户 %d 的 %d 条游戏存档\n", userID, deletedCount)

	// 重置游戏引擎中的会话状态（如果存在）
	if gameController != nil && stateManager != nil {
		// 清除内存中的会话数据
		playerIDStr := fmt.Sprintf("%d", userID)
		err := stateManager.DeletePlayerSessions(playerIDStr)
		if err != nil {
			fmt.Printf("⚠️ 清除内存会话数据失败: %v\n", err)
		} else {
			fmt.Printf("✅ 清除用户 %d 的内存会话数据\n", userID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "机缘已重启，所有存档已清空",
		"deleted_saves": deletedCount,
	})
}
