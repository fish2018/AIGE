package game_engine

import (
	"AIGE/services"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"
)

// GameController handles game logic and AI interactions
type GameController struct {
	modLoader          *ModLoader
	stateManager       *StateManager
	aiClient           *services.AIClient
	compressionManager *CompressionManager // 新增
	// AI配置（从数据库或配置文件加载）
	defaultProvider AIProvider
}

// AIProvider 表示AI提供商配置
type AIProvider struct {
	APIType string
	BaseURL string
	APIKey  string
	ModelID string
}

// NewGameController creates a new game controller
func NewGameController(modLoader *ModLoader, stateManager *StateManager) *GameController {
	aiClient := services.NewAIClient()
	compressionManager := NewCompressionManager(aiClient, stateManager)
	
	gc := &GameController{
		modLoader:          modLoader,
		stateManager:       stateManager,
		aiClient:           aiClient,
		compressionManager: compressionManager,
		// 默认配置，应该从数据库或环境变量加载
		defaultProvider: AIProvider{
			APIType: "openai",
			BaseURL: "https://api.openai.com",
			APIKey:  "", // 需要配置
			ModelID: "gpt-4o-mini",
		},
	}
	
	// 设置压缩管理器的GameController引用
	compressionManager.SetGameController(gc)
	
	return gc
}

// SetAIProvider 设置AI提供商配置
func (gc *GameController) SetAIProvider(provider AIProvider) {
	gc.defaultProvider = provider
}

// InitializeGame initializes a new game session for a player or loads existing save
func (gc *GameController) InitializeGame(playerID, modID string) (*GameSession, error) {
	// Load the mod
	mod, err := gc.modLoader.LoadMod(modID)
	if err != nil {
		return nil, fmt.Errorf("failed to load mod: %w", err)
	}

	// Try to load existing session first
	existingSession, err := gc.stateManager.GetSession(playerID, modID)
	if err == nil {
		// Session exists, return it (daily reset already handled in GetSession)
		fmt.Printf("[GameController] 加载已存在的存档: 玩家=%s, mod=%s\n", playerID, modID)
		return existingSession, nil
	}

	// No existing session, create a new one
	fmt.Printf("[GameController] 创建新存档: 玩家=%s, mod=%s\n", playerID, modID)

	// Create initial state from mod config
	initialState := make(map[string]interface{})
	for k, v := range mod.Config.InitialState {
		initialState[k] = v
	}

	// Get system prompt
	systemPrompt := mod.Prompts["game_master"]

	// Create session
	session, err := gc.stateManager.CreateSession(playerID, modID, initialState, systemPrompt)
	if err != nil {
		return nil, err
	}

	// Add welcome message to display history
	session.DisplayHistory = append(session.DisplayHistory, mod.Config.WelcomeMessage)

	// Save session
	if err := gc.stateManager.SaveSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

// StartTrial starts a new trial/game round
func (gc *GameController) StartTrial(playerID, modID string) error {
	session, err := gc.stateManager.GetSession(playerID, modID)
	if err != nil {
		return err
	}

	mod, err := gc.modLoader.GetMod(session.ModID)
	if err != nil {
		return err
	}

	// Check if player has opportunities remaining
	opps, _ := session.State["opportunities_remaining"].(float64)
	if opps <= 0 {
		return fmt.Errorf("no opportunities remaining")
	}

	// Mark as processing
	session.State["is_processing"] = true
	gc.stateManager.SaveSession(session)

	// Get start trial prompt
	startPrompt := mod.Prompts["start_trial"]
	if startPrompt == "" {
		startPrompt = mod.Prompts["start_game"]
	}

	// Call AI to generate initial scenario
	aiResponse, err := gc.callAI(session, startPrompt, mod)
	if err != nil {
		session.State["is_processing"] = false
		gc.stateManager.SaveSession(session)
		return err
	}

	// Parse and apply response
	if err := gc.parseAndApplyAIResponse(session, aiResponse, mod, ""); err != nil {
		session.State["is_processing"] = false
		gc.stateManager.SaveSession(session)
		return err
	}

	// Mark as not processing
	session.State["is_processing"] = false
	gc.stateManager.SaveSession(session)

	return nil
}

// ProcessAction processes a player's action
func (gc *GameController) ProcessAction(playerID, modID, action string) error {
	session, err := gc.stateManager.GetSession(playerID, modID)
	if err != nil {
		return err
	}

	mod, err := gc.modLoader.GetMod(session.ModID)
	if err != nil {
		return err
	}

	// Check if already processing
	if isProcessing, ok := session.State["is_processing"].(bool); ok && isProcessing {
		return fmt.Errorf("already processing an action")
	}

	// Mark as processing
	session.State["is_processing"] = true
	gc.stateManager.SaveSession(session)

	// Note: User action is already added to display_history by frontend for immediate display
	// Only add to internal history for AI context
	userMsg := Message{
		Role:      "user",
		Content:   action,
		Timestamp: time.Now(),
	}
	session.RecentHistory = append(session.RecentHistory, userMsg)

	// Call AI
	currentStateJSON, _ := json.Marshal(session.State)
	prompt := fmt.Sprintf("%s\n\n当前游戏状态：\n%s", action, string(currentStateJSON))

	aiResponse, err := gc.callAI(session, prompt, mod)
	if err != nil {
		session.State["is_processing"] = false
		gc.stateManager.SaveSession(session)
		return err
	}

	// Parse and apply response
	if err := gc.parseAndApplyAIResponse(session, aiResponse, mod, action); err != nil {
		session.State["is_processing"] = false
		gc.stateManager.SaveSession(session)
		return err
	}

	// Mark as not processing
	session.State["is_processing"] = false
	gc.stateManager.SaveSession(session)

	return nil
}

// buildAIMessages builds AI messages using new compression system
func (gc *GameController) buildAIMessages(session *GameSession, userPrompt string, mod *GameMod) []services.Message {
	messages := []services.Message{}
	
	// 1. 动态加载最新系统提示词
	messages = append(messages, services.Message{
		Role:    "system",
		Content: mod.Prompts["game_master"], // 始终使用最新版本
	})
	
	// 2. 添加压缩摘要（如果存在）
	if session.CompressedSummary != "" {
		fmt.Printf("[消息构建] 添加压缩摘要，长度: %d 字符\n", len(session.CompressedSummary))
		messages = append(messages, services.Message{
			Role:    "system",
			Content: fmt.Sprintf("【历史摘要】%s", session.CompressedSummary),
		})
	} else {
		fmt.Printf("[消息构建] 无压缩摘要\n")
	}
	
	// 3. 添加最近4条对话
	fmt.Printf("[消息构建] 添加最近历史记录: %d 条\n", len(session.RecentHistory))
	for _, msg := range session.RecentHistory {
		messages = append(messages, services.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// 4. 添加当前用户输入
	messages = append(messages, services.Message{
		Role:    "user",
		Content: userPrompt,
	})
	
	fmt.Printf("[消息构建] 总消息数: %d (系统提示词 + 摘要 + 历史记录 + 当前输入)\n", len(messages))
	
	return messages
}

// callAI calls the AI service
func (gc *GameController) callAI(session *GameSession, prompt string, mod *GameMod) (string, error) {
	// 使用新的消息构建方法
	messages := gc.buildAIMessages(session, prompt, mod)

	// Check if AI provider is configured
	if gc.defaultProvider.APIKey == "" {
		return "", fmt.Errorf("AI provider not configured - please set API key in admin panel")
	}

	// Call AI service based on provider type
	var response interface{}
	var err error

	switch gc.defaultProvider.APIType {
	case "openai":
		response, err = gc.aiClient.CallOpenAI(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			false, // non-streaming for game logic
		)
	case "anthropic":
		response, err = gc.aiClient.CallAnthropic(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			false,
		)
	case "google":
		response, err = gc.aiClient.CallGoogle(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			false,
		)
	default:
		return "", fmt.Errorf("unsupported API type: %s", gc.defaultProvider.APIType)
	}

	if err != nil {
		return "", fmt.Errorf("AI call failed: %w", err)
	}

	// Extract content from response
	if respMap, ok := response.(map[string]interface{}); ok {
		if content, ok := respMap["content"].(string); ok {
			return content, nil
		}
	}

	return "", fmt.Errorf("invalid AI response format")
}

// parseAndApplyAIResponse parses AI response and applies state updates
func (gc *GameController) parseAndApplyAIResponse(session *GameSession, aiResponse string, mod *GameMod, originalAction string) error {
	// Extract narrative from new format ($...$)
	narrativeFromFormat := extractNarrative(aiResponse)
	narrative := narrativeFromFormat
	// narrative = strings.ReplaceAll(narrative, "$", "")

	// Extract JSON from response (@...@)
	jsonStr := extractJSON(aiResponse)
	if jsonStr == "" {
		return fmt.Errorf("no valid JSON found in AI response")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return fmt.Errorf("failed to parse AI response JSON: %w", err)
	}

	// Add AI response to history
	session.RecentHistory = append(session.RecentHistory, Message{
		Role:      "assistant",
		Content:   aiResponse,
		Timestamp: time.Now(),
	})

	// Get narrative - prefer format over JSON
	if narrative == "" {
		narrative, _ = parsed["narrative"].(string)
		// Clean any $ symbols that shouldn't be in the final narrative
		narrative = strings.ReplaceAll(narrative, "$", "")
	}

	// Check if this is a roll request (two-stage judgment)
	if rollRequest, hasRoll := parsed["roll_request"].(map[string]interface{}); hasRoll {
		// Add narrative to display
		if narrative != "" {
			session.DisplayHistory = append(session.DisplayHistory, narrative)
			gc.stateManager.SaveSession(session)
		}

		// Execute roll
		rollResult := gc.executeRoll(rollRequest, mod)

		// TODO: Send roll event to frontend via WebSocket

		// Request AI to continue based on roll result
		rollResultText := fmt.Sprintf("【判定结果：%s】", rollResult["outcome"])
		session.DisplayHistory = append(session.DisplayHistory, rollResultText)

		// Call AI again with roll result
		currentStateJSON, _ := json.Marshal(session.State)
		prompt := fmt.Sprintf("%s\n\n请基于此判定结果继续叙事。当前状态：\n%s", rollResultText, string(currentStateJSON))

		aiResponse2, err := gc.callAI(session, prompt, mod)
		if err != nil {
			return err
		}

		// Parse second response with new format
		narrativeFromFormat2 := extractNarrative(aiResponse2)
		jsonStr2 := extractJSON(aiResponse2)
		var parsed2 map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr2), &parsed2); err != nil {
			return fmt.Errorf("failed to parse second AI response: %w", err)
		}

		// Add second response to history
		aiMsg2 := Message{
			Role:      "assistant",
			Content:   aiResponse2,
			Timestamp: time.Now(),
		}
		session.RecentHistory = append(session.RecentHistory, aiMsg2)

		// Get second narrative - prefer format over JSON
		narrative2 := narrativeFromFormat2
		if narrative2 == "" {
			narrative2, _ = parsed2["narrative"].(string)
		}
		// narrative2 = strings.ReplaceAll(narrative2, "$", "")
		if narrative2 != "" {
			session.DisplayHistory = append(session.DisplayHistory, narrative2)
		}

		// Apply state update from second response
		if stateUpdate, ok := parsed2["state_update"].(map[string]interface{}); ok {
			ApplyStateUpdate(session.State, stateUpdate)
		}

	} else {
		// No roll request, direct state update
		if narrative != "" {
			session.DisplayHistory = append(session.DisplayHistory, narrative)
		}

		// Apply state update
		if stateUpdate, ok := parsed["state_update"].(map[string]interface{}); ok {
			ApplyStateUpdate(session.State, stateUpdate)

			// Check for special program triggers
			if trigger, hasTrigger := stateUpdate["trigger_program"].(map[string]interface{}); hasTrigger {
				gc.handleProgramTrigger(session, trigger, mod)
			}
		}
	}

	// Save session
	return gc.stateManager.SaveSession(session)
}

// executeRoll executes a dice roll
func (gc *GameController) executeRoll(rollRequest map[string]interface{}, mod *GameMod) map[string]interface{} {
	rollType, _ := rollRequest["type"].(string)
	target, _ := rollRequest["target"].(float64)
	sides, _ := rollRequest["sides"].(float64)

	if sides == 0 {
		sides = float64(mod.Config.GameConfig.RollSettings.DefaultSides)
	}

	// Execute roll
	result := rand.Intn(int(sides)) + 1

	// Determine outcome
	var outcome string
	critSuccess := mod.Config.GameConfig.RollSettings.CriticalSuccessThreshold
	critFail := mod.Config.GameConfig.RollSettings.CriticalFailureThreshold

	if float64(result) <= sides*critSuccess {
		outcome = "大成功"
	} else if float64(result) <= target {
		outcome = "成功"
	} else if float64(result) >= sides*critFail {
		outcome = "大失败"
	} else {
		outcome = "失败"
	}

	// Determine success based on outcome
	success := outcome == "成功" || outcome == "大成功"

	return map[string]interface{}{
		"type":    rollType,
		"target":  target,
		"sides":   sides,
		"result":  result,
		"outcome": outcome,
		"success": success,
	}
}

// handleProgramTrigger handles special program triggers (like ending the game)
func (gc *GameController) handleProgramTrigger(session *GameSession, trigger map[string]interface{}, mod *GameMod) {
	triggerName, _ := trigger["name"].(string)

	switch triggerName {
	case "spiritStoneConverter":
		// Handle game end and reward calculation
		spiritStones, _ := trigger["spirit_stones"].(float64)
		reward := gc.calculateReward(int(spiritStones), mod)

		// Mark as completed
		session.State["daily_success_achieved"] = true
		session.State["is_in_trial"] = false

		// Add completion message
		message := fmt.Sprintf("\n\n【天机阁长老】：道友功德圆满！获得修行资源：%d", reward)
		session.DisplayHistory = append(session.DisplayHistory, message)
	}
}

// calculateReward calculates reward based on spirit stones (diminishing returns)
func (gc *GameController) calculateReward(spiritStones int, mod *GameMod) int {
	if spiritStones <= 0 {
		return 0
	}

	scalingFactor := float64(mod.Config.GameConfig.RewardScalingFactor)
	// Diminishing returns formula: reward = scaling * min(30, max(1, 3 * (stones^(1/6))))
	value := 3.0 * pow(float64(spiritStones), 1.0/6.0)
	if value < 1.0 {
		value = 1.0
	}
	if value > 30.0 {
		value = 30.0
	}

	return int(scalingFactor * value)
}

// extractNarrative extracts narrative text from AI response (handles $...$ format)
func extractNarrative(response string) string {
	// Remove think tags first
	if strings.Contains(response, "<think>") && strings.Contains(response, "</think>") {
		endIdx := strings.LastIndex(response, "</think>")
		response = response[endIdx+8:]
	}

	// Handle format: $...$ for narrative
	if strings.Contains(response, "$") {
		startIdx := strings.Index(response, "$")
		if startIdx >= 0 {
			endIdx := strings.Index(response[startIdx+1:], "$")
			if endIdx >= 0 {
				narrativeContent := response[startIdx+1 : startIdx+1+endIdx]
				return narrativeContent
			}
		}
	}

	// Fallback: try to extract narrative from JSON if no $ delimiters
	return ""
}

// extractJSON extracts JSON from AI response (handles format with $ and @ delimiters)
func extractJSON(response string) string {
	// Remove think tags first
	if strings.Contains(response, "<think>") && strings.Contains(response, "</think>") {
		endIdx := strings.LastIndex(response, "</think>")
		response = response[endIdx+8:]
	}

	// Handle format: @...@ for JSON
	if strings.Contains(response, "@") {
		startIdx := strings.Index(response, "@")
		if startIdx >= 0 {
			endIdx := strings.Index(response[startIdx+1:], "@")
			if endIdx >= 0 {
				jsonContent := response[startIdx+1 : startIdx+1+endIdx]
				jsonContent = strings.TrimSpace(jsonContent)
				// Validate it's JSON by checking if it starts with {
				if strings.HasPrefix(jsonContent, "{") {
					return jsonContent
				}
			}
		}
	}

	// Handle markdown code blocks (fallback)
	if strings.Contains(response, "```json") {
		startIdx := strings.Index(response, "```json")
		if startIdx >= 0 {
			startIdx += 7 // Skip "```json"
			endIdx := strings.Index(response[startIdx:], "```")
			if endIdx >= 0 {
				jsonContent := response[startIdx : startIdx+endIdx]
				return strings.TrimSpace(jsonContent)
			}
		}
	}

	// Handle single backticks around JSON (fallback)
	if strings.Contains(response, "`{") && strings.Contains(response, "}`") {
		startIdx := strings.Index(response, "`{")
		endIdx := strings.LastIndex(response, "}`")
		if startIdx >= 0 && endIdx >= 0 && endIdx > startIdx {
			jsonContent := response[startIdx+1 : endIdx+1] // Skip the backtick
			return strings.TrimSpace(jsonContent)
		}
	}

	// Find JSON without delimiters (fallback)
	response = strings.TrimSpace(response)

	// Try to extract JSON block
	if startIdx := strings.Index(response, "{"); startIdx >= 0 {
		if endIdx := strings.LastIndex(response, "}"); endIdx >= 0 {
			return response[startIdx : endIdx+1]
		}
	}

	return ""
}

// pow is a simple power function
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp*100); i++ {
		result *= base
	}
	return result
}

// StreamCallback 流式输出回调函数类型
type StreamCallback func(chunk string) error

// RollEventCallback 判定事件回调函数类型
type RollEventCallback func(rollEvent map[string]interface{}) error

// ProcessActionStream processes a player action with streaming narrative
func (gc *GameController) ProcessActionStream(playerID, modID, action string, streamCallback StreamCallback, rollCallback RollEventCallback, secondStageCallback StreamCallback) error {
	session, err := gc.stateManager.GetSession(playerID, modID)
	if err != nil {
		return err
	}

	mod, err := gc.modLoader.GetMod(modID)
	if err != nil {
		return err
	}

	// Check if already processing
	if isProcessing, ok := session.State["is_processing"].(bool); ok && isProcessing {
		return fmt.Errorf("已有操作正在处理中")
	}

	session.State["is_processing"] = true
	gc.stateManager.SaveSession(session)

	// Note: User action is already added to display_history by frontend for immediate display
	// Only add to internal history for AI context
	userMsg := Message{
		Role:      "user",
		Content:   action,
		Timestamp: time.Now(),
	}
	session.RecentHistory = append(session.RecentHistory, userMsg)

	var prompt string

	// Handle special actions
	if action == "start_trial" {
		// Use start trial prompt
		startPrompt := mod.Prompts["start_trial"]
		if startPrompt == "" {
			startPrompt = mod.Prompts["start_game"]
		}
		prompt = startPrompt
	} else {
		// Build regular AI prompt
		currentStateJSON, _ := json.Marshal(session.State)
		prompt = fmt.Sprintf("%s\n\n当前游戏状态：\n%s", action, string(currentStateJSON))
	}

	// Call AI with streaming (with retry mechanism)
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("[一阶段重试] 尝试第 %d/%d 次调用AI\n", attempt, maxRetries)

		err = gc.callAIStream(session, prompt, mod, action, streamCallback, rollCallback, secondStageCallback)
		if err == nil {
			fmt.Printf("[一阶段重试] 第 %d 次调用成功\n", attempt)
			break
		}

		lastErr = err
		fmt.Printf("[一阶段重试] 第 %d 次调用失败: %v\n", attempt, err)

		// 检查是否是JSON格式错误
		if strings.Contains(err.Error(), "no valid JSON found") ||
			strings.Contains(err.Error(), "failed to parse") {
			fmt.Printf("[一阶段重试] 检测到JSON格式错误，准备重试...\n")

			if attempt < maxRetries {
				// 在重试前稍等一下，避免请求过于频繁
				// time.Sleep(time.Millisecond * 500)

				// 修改prompt，要求AI更加注意格式
				if action == "start_new_trial" {
					// 新游戏开始，使用特殊提示
					currentStateJSON, _ := json.Marshal(session.State)
					prompt = fmt.Sprintf("%s\n\n⚠️ 重要格式要求：\n1. 必须严格按照JSON格式输出\n2. 确保JSON语法正确，特别注意引号和逗号\n3. 所有字符串值都要用双引号包围\n4. 叙事内容在JSON的narrative字段中\n\n当前游戏状态：\n%s", mod.Prompts["start_game"], string(currentStateJSON))
				} else {
					// 常规动作，添加格式提醒
					currentStateJSON, _ := json.Marshal(session.State)
					prompt = fmt.Sprintf("%s\n\n⚠️ 重要格式要求：\n1. 必须严格按照JSON格式输出\n2. 确保JSON语法正确，特别注意引号和逗号\n3. 所有字符串值都要用双引号包围\n4. 叙事内容在JSON的narrative字段中\n\n当前游戏状态：\n%s", action, string(currentStateJSON))
				}
			}
		} else {
			// 非格式错误，不重试
			fmt.Printf("[一阶段重试] 非格式错误，不进行重试: %v\n", err)
			break
		}
	}

	if err != nil {
		fmt.Printf("[一阶段重试] 所有重试均失败，最后错误: %v\n", lastErr)
		session.State["is_processing"] = false
		gc.stateManager.SaveSession(session)
		return fmt.Errorf("first stage AI call failed after %d attempts: %w", maxRetries, lastErr)
	}

	session.State["is_processing"] = false
	gc.stateManager.SaveSession(session)

	return err
}

// callAIStream calls AI service with streaming support
func (gc *GameController) callAIStream(session *GameSession, prompt string, mod *GameMod, originalAction string, streamCallback StreamCallback, rollCallback RollEventCallback, secondStageCallback StreamCallback) error {
	// 使用新的消息构建方法
	messages := gc.buildAIMessages(session, prompt, mod)

	// 调试：打印发送给AI的消息
	fmt.Printf("\n=== 发送给AI的消息 (%d条) ===\n", len(messages))
	for i, msg := range messages {
		contentPreview := msg.Content
		if len(contentPreview) > 200 {
			contentPreview = contentPreview[:200] + "...(总长:" + fmt.Sprintf("%d", len(msg.Content)) + ")"
		}
		fmt.Printf("[%d] %s: %s\n", i, msg.Role, contentPreview)
	}
	fmt.Printf("=== 消息结束 ===\n\n")

	// Check if AI provider is configured
	if gc.defaultProvider.APIKey == "" {
		return fmt.Errorf("AI provider not configured")
	}

	fmt.Printf("使用AI提供商: %s, 模型: %s\n", gc.defaultProvider.APIType, gc.defaultProvider.ModelID)

	// Call AI service with streaming
	var response interface{}
	var err error

	switch gc.defaultProvider.APIType {
	case "openai":
		response, err = gc.aiClient.CallOpenAI(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			true, // streaming
		)
	case "anthropic":
		response, err = gc.aiClient.CallAnthropic(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			true,
		)
	case "google":
		response, err = gc.aiClient.CallGoogle(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			true,
		)
	default:
		return fmt.Errorf("unsupported API type: %s", gc.defaultProvider.APIType)
	}

	if err != nil {
		return fmt.Errorf("AI call failed: %w", err)
	}

	// Process stream
	body, ok := response.(io.ReadCloser)
	if !ok {
		return fmt.Errorf("invalid stream response")
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 128*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var fullResponse strings.Builder
	var narrativeBuffer strings.Builder
	var jsonStarted bool

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		chunk := gc.aiClient.ParseStreamChunk(gc.defaultProvider.APIType, line)
		if chunk != nil {
			if content, ok := chunk["content"].(string); ok && content != "" {
				fullResponse.WriteString(content)

				if !jsonStarted {
					// 检测是否遇到 @ 标记（新格式）或其他JSON标记
					if strings.Contains(content, "@") || strings.Contains(content, "```json") || strings.Contains(content, "{") {
						jsonStarted = true
						// 发送JSON标记之前的内容
						beforeJson := content
						if atMarkIndex := strings.Index(content, "@"); atMarkIndex >= 0 {
							beforeJson = content[:atMarkIndex]
						} else if jsonMarkIndex := strings.Index(content, "```json"); jsonMarkIndex >= 0 {
							beforeJson = content[:jsonMarkIndex]
						} else if jsonIndex := strings.Index(content, "{"); jsonIndex >= 0 {
							beforeJson = content[:jsonIndex]
						}

						if strings.TrimSpace(beforeJson) != "" {
							narrativeBuffer.WriteString(beforeJson)
							if err := streamCallback(beforeJson); err != nil {
								return err
							}
						}
					} else {
						// 纯narrative内容，直接发送
						content = strings.ReplaceAll(content, "$", "")
						narrativeBuffer.WriteString(content)
						if err := streamCallback(content); err != nil {
							return err
						}
					}
				}
				// JSON部分不再流式发送
			}

			if done, ok := chunk["done"].(bool); ok && done {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Parse and apply the complete response
	aiResponse := fullResponse.String()

	fmt.Printf("\n=== AI完整响应 ===\n%s\n=== 响应结束 ===\n", aiResponse)

	// Parse the response to check for roll_request
	jsonStr := extractJSON(aiResponse)
	if jsonStr == "" {
		fmt.Printf("ERROR: 无法从AI响应中提取JSON\n")
		fmt.Printf("完整响应: %s\n", aiResponse)
		fmt.Printf("响应长度: %d 字符\n", len(aiResponse))
		return fmt.Errorf("no valid JSON found in AI response")
	}

	fmt.Printf("\n=== 提取的JSON ===\n%s\n=== JSON结束 ===\n", jsonStr)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		fmt.Printf("DEBUG: Failed to parse JSON. Extracted JSON: %s\n", jsonStr)
		fmt.Printf("DEBUG: Full AI response: %s\n", aiResponse)
		return fmt.Errorf("failed to parse AI response JSON: %w", err)
	}

	// Add to history and handle compression
	aiMsg := Message{
		Role:      "assistant",
		Content:   aiResponse,
		Timestamp: time.Now(),
	}
	
	// 获取最后一条用户消息
	var lastUserMsg Message
	if len(session.RecentHistory) > 0 {
		lastUserMsg = session.RecentHistory[len(session.RecentHistory)-1]
	}
	
	// 处理对话历史压缩
	gc.compressionManager.ProcessNewMessage(session, lastUserMsg, aiMsg)

	// Check if this is a roll request (two-stage judgment)
	if rollRequest, hasRoll := parsed["roll_request"].(map[string]interface{}); hasRoll {
		// Execute roll
		rollResult := gc.executeRoll(rollRequest, mod)

		// Send roll event to frontend
		rollEvent := map[string]interface{}{
			"type":        rollRequest["type"],
			"target":      rollRequest["target"],
			"description": rollRequest["description"],
			"result":      rollResult["result"],
			"outcome":     rollResult["outcome"],
			"success":     rollResult["success"],
		}

		// Send roll event to frontend
		if rollCallback != nil {
			if err := rollCallback(rollEvent); err != nil {
				return err
			}
		}

		// Send roll result as separate message via streaming
		rollResultText := fmt.Sprintf("【判定结果：%s】", rollResult["outcome"])
		if err := streamCallback(rollResultText); err != nil {
			return err
		}

		// Call AI again with roll result for second stage
		currentStateJSON, _ := json.Marshal(session.State)
		prompt := fmt.Sprintf("判定已完成：%s\n\n请基于此判定结果继续叙事。重要提醒：\n1. 不要重复输出判定结果\n2. 不要重复之前的叙事内容\n3. 只输出基于判定结果的后续新情节\n\n当前状态：\n%s", rollResult["outcome"], string(currentStateJSON))

		// Get first narrative for comparison - prefer format over JSON
		firstNarrativeFromFormat := extractNarrative(aiResponse)
		firstNarrative := firstNarrativeFromFormat
		if firstNarrative == "" {
			firstNarrative, _ = parsed["narrative"].(string)
		}

		// Second stage AI call (streaming) with retry mechanism
		maxRetries := 3
		var lastErr error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			fmt.Printf("[二阶段重试] 尝试第 %d/%d 次调用AI\n", attempt, maxRetries)

			err = gc.callAIStreamSecondStage(session, prompt, mod, firstNarrative, secondStageCallback)
			if err == nil {
				fmt.Printf("[二阶段重试] 第 %d 次调用成功\n", attempt)
				break
			}

			lastErr = err
			fmt.Printf("[二阶段重试] 第 %d 次调用失败: %v\n", attempt, err)

			// 检查是否是JSON格式错误
			if strings.Contains(err.Error(), "no valid JSON found") ||
				strings.Contains(err.Error(), "failed to parse") {
				fmt.Printf("[二阶段重试] 检测到JSON格式错误，准备重试...\n")

				if attempt < maxRetries {
					// 在重试前稍等一下，避免请求过于频繁
					// time.Sleep(time.Millisecond * 500)

					// 修改prompt，要求AI更加注意格式
					prompt = fmt.Sprintf("判定已完成：%s\n\n请基于此判定结果继续叙事。\n\n⚠️ 重要格式要求：\n1. 不要重复输出判定结果\n2. 不要重复之前的叙事内容\n3. 只输出基于判定结果的后续新情节\n4. 必须严格按照JSON格式输出，确保JSON语法正确\n5. 叙事内容在JSON中，不要在JSON外输出额外内容\n\n当前状态：\n%s", rollResult["outcome"], string(currentStateJSON))
				}
			} else {
				// 非格式错误，不重试
				fmt.Printf("[二阶段重试] 非格式错误，不进行重试: %v\n", err)
				break
			}
		}

		if err != nil {
			fmt.Printf("[二阶段重试] 所有重试均失败，最后错误: %v\n", lastErr)
			return fmt.Errorf("second stage AI call failed after %d attempts: %w", maxRetries, lastErr)
		}

	} else {
		// No roll request, direct state update
		if narrative, ok := parsed["narrative"].(string); ok && narrative != "" {
			// Note: Don't add to DisplayHistory here, frontend handles it via streaming
		}

		// Apply state update
		if stateUpdate, ok := parsed["state_update"].(map[string]interface{}); ok {
			ApplyStateUpdate(session.State, stateUpdate)

			// Check if trial ended (game over)
			if isInTrial, exists := stateUpdate["is_in_trial"]; exists {
				if inTrial, ok := isInTrial.(bool); ok && !inTrial {
					// Trial ended, immediately stop processing
					session.State["is_processing"] = false
					fmt.Printf("DEBUG: Trial ended, setting is_processing = false\n")
				}
			}

			// Check for special program triggers
			if trigger, hasTrigger := stateUpdate["trigger_program"].(map[string]interface{}); hasTrigger {
				gc.handleProgramTrigger(session, trigger, mod)
			}
		}
	}

	// Save session
	return gc.stateManager.SaveSession(session)
}

// filterDuplicateContent filters out duplicate content from the second narrative
func filterDuplicateContent(secondNarrative, firstNarrative string) string {
	// If first narrative is empty, return second narrative as is
	if firstNarrative == "" {
		return secondNarrative
	}

	// Clean up the narratives
	secondNarrative = strings.TrimSpace(secondNarrative)
	firstNarrative = strings.TrimSpace(firstNarrative)

	// Simple approach: if second narrative starts with first narrative,
	// return only the part after first narrative
	if strings.HasPrefix(secondNarrative, firstNarrative) {
		remaining := strings.TrimPrefix(secondNarrative, firstNarrative)
		return strings.TrimSpace(remaining)
	}

	// Split both narratives into sentences for better comparison
	firstSentences := strings.Split(firstNarrative, "。")
	secondSentences := strings.Split(secondNarrative, "。")

	// Find where the unique content starts in second narrative
	uniqueStartIndex := len(secondSentences) // Default to end if no unique content found

	for i, sentence := range secondSentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// Check if this sentence exists in first narrative
		found := false
		for _, firstSentence := range firstSentences {
			firstSentence = strings.TrimSpace(firstSentence)
			if firstSentence != "" && strings.Contains(sentence, firstSentence) || strings.Contains(firstSentence, sentence) {
				found = true
				break
			}
		}

		if !found {
			uniqueStartIndex = i
			break
		}
	}

	// Return unique sentences
	if uniqueStartIndex < len(secondSentences) {
		uniqueSentences := secondSentences[uniqueStartIndex:]
		result := strings.Join(uniqueSentences, "。")
		return strings.TrimSpace(result)
	}

	// If no unique content found, return the second narrative as is
	// (this might happen if AI generates completely new content)
	return secondNarrative
}

// callAIStreamSecondStage calls AI service for second stage with streaming support
func (gc *GameController) callAIStreamSecondStage(session *GameSession, prompt string, mod *GameMod, firstNarrative string, secondStageCallback StreamCallback) error {
	// Build messages from session history (which already contains system prompt)
	messages := gc.buildAIMessages(session, prompt, mod)

	// Check if AI provider is configured
	if gc.defaultProvider.APIKey == "" {
		return fmt.Errorf("AI provider not configured")
	}

	// Call AI service with streaming
	var response interface{}
	var err error

	switch gc.defaultProvider.APIType {
	case "openai":
		response, err = gc.aiClient.CallOpenAI(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			true, // streaming
		)
	case "anthropic":
		response, err = gc.aiClient.CallAnthropic(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			true,
		)
	case "google":
		response, err = gc.aiClient.CallGoogle(
			gc.defaultProvider.BaseURL,
			gc.defaultProvider.APIKey,
			gc.defaultProvider.ModelID,
			messages,
			true,
		)
	default:
		return fmt.Errorf("unsupported API type: %s", gc.defaultProvider.APIType)
	}

	if err != nil {
		return fmt.Errorf("AI call failed: %w", err)
	}

	// Process stream
	body, ok := response.(io.ReadCloser)
	if !ok {
		return fmt.Errorf("invalid stream response")
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 128*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var fullResponse strings.Builder
	var jsonStarted bool

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		chunk := gc.aiClient.ParseStreamChunk(gc.defaultProvider.APIType, line)
		if chunk != nil {
			if content, ok := chunk["content"].(string); ok && content != "" {
				fullResponse.WriteString(content)

				if !jsonStarted {
					// 检测是否遇到 @ 标记（新格式）或其他JSON标记
					if strings.Contains(content, "@") || strings.Contains(content, "```json") || strings.Contains(content, "{") {
						jsonStarted = true
						// 发送JSON标记之前的内容
						beforeJson := content
						if atMarkIndex := strings.Index(content, "@"); atMarkIndex >= 0 {
							beforeJson = content[:atMarkIndex]
						} else if jsonMarkIndex := strings.Index(content, "```json"); jsonMarkIndex >= 0 {
							beforeJson = content[:jsonMarkIndex]
						} else if jsonIndex := strings.Index(content, "{"); jsonIndex >= 0 {
							beforeJson = content[:jsonIndex]
						}

						if strings.TrimSpace(beforeJson) != "" {
							if err := secondStageCallback(beforeJson); err != nil {
								return err
							}
						}
					} else {
						// 纯narrative内容，直接发送
						content = strings.ReplaceAll(content, "$", "")
						if err := secondStageCallback(content); err != nil {
							return err
						}
					}
				}
				// JSON部分不再流式发送
			}

			if done, ok := chunk["done"].(bool); ok && done {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Parse and apply the complete response
	aiResponse := fullResponse.String()

	fmt.Printf("\n=== 第二阶段AI完整响应 ===\n%s\n=== 响应结束 ===\n", aiResponse)

	// Parse the response
	jsonStr := extractJSON(aiResponse)
	if jsonStr == "" {
		fmt.Printf("ERROR: 无法从第二阶段AI响应中提取JSON\n")
		fmt.Printf("完整响应: %s\n", aiResponse)
		fmt.Printf("响应长度: %d 字符\n", len(aiResponse))
		return fmt.Errorf("no valid JSON found in second AI response")
	}

	fmt.Printf("\n=== 第二阶段提取的JSON ===\n%s\n=== JSON结束 ===\n", jsonStr)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		fmt.Printf("DEBUG: Failed to parse second JSON. Extracted JSON: %s\n", jsonStr)
		fmt.Printf("DEBUG: Full second AI response: %s\n", aiResponse)
		return fmt.Errorf("failed to parse second AI response JSON: %w", err)
	}

	// Add to history and handle compression
	aiMsg := Message{
		Role:      "assistant",
		Content:   aiResponse,
		Timestamp: time.Now(),
	}
	session.RecentHistory = append(session.RecentHistory, aiMsg)

	// Apply state update
	if stateUpdate, ok := parsed["state_update"].(map[string]interface{}); ok {
		ApplyStateUpdate(session.State, stateUpdate)

		// Check if trial ended (game over) in second response
		if isInTrial, exists := stateUpdate["is_in_trial"]; exists {
			if inTrial, ok := isInTrial.(bool); ok && !inTrial {
				// Trial ended, immediately stop processing
				session.State["is_processing"] = false
				fmt.Printf("DEBUG: Trial ended in second response, setting is_processing = false\n")
			}
		}

		// Check for special program triggers
		if trigger, hasTrigger := stateUpdate["trigger_program"].(map[string]interface{}); hasTrigger {
			gc.handleProgramTrigger(session, trigger, mod)
		}
	}

	return nil
}
