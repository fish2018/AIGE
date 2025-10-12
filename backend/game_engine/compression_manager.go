package game_engine

import (
	"AIGE/services"
	"fmt"
	"strings"
)

type CompressionManager struct {
	aiClient           *services.AIClient
	stateManager       *StateManager    // 需要保存到数据库
	gameController     *GameController  // 新增：获取AI配置
	compressionInterval int // 15轮压缩一次
	maxRecentHistory   int // 保留最近4条
}

func NewCompressionManager(aiClient *services.AIClient, stateManager *StateManager) *CompressionManager {
	return &CompressionManager{
		aiClient:           aiClient,
		stateManager:       stateManager,
		gameController:     nil, // 稍后通过SetGameController设置
		compressionInterval: 15,
		maxRecentHistory:   4,
	}
}

// SetGameController 设置GameController引用以获取AI配置
func (cm *CompressionManager) SetGameController(gc *GameController) {
	cm.gameController = gc
}

func (cm *CompressionManager) ProcessNewMessage(session *GameSession, userMsg, aiMsg Message) {
	// 添加新对话到recent history
	session.RecentHistory = append(session.RecentHistory, userMsg, aiMsg)
	
	fmt.Printf("[压缩检查] 当前历史记录数: %d, 压缩阈值: %d\n", len(session.RecentHistory), cm.compressionInterval)
	
	// 检查是否需要压缩
	if len(session.RecentHistory) >= cm.compressionInterval {
		fmt.Printf("[压缩触发] 开始压缩历史记录...\n")
		cm.compressAndCleanup(session)
	}
}

func (cm *CompressionManager) compressAndCleanup(session *GameSession) {
	// 准备压缩内容（保留最近4条，压缩其余）
	toCompress := session.RecentHistory[:len(session.RecentHistory)-cm.maxRecentHistory]
	toKeep := session.RecentHistory[len(session.RecentHistory)-cm.maxRecentHistory:]
	
	fmt.Printf("[压缩详情] 待压缩消息数: %d, 保留消息数: %d\n", len(toCompress), len(toKeep))
	
	// 构建压缩提示词
	compressionPrompt := cm.buildCompressionPrompt(toCompress)
	
	// 异步压缩
	go func() {
		fmt.Printf("[压缩进行] 调用AI进行压缩...\n")
		newSummary, err := cm.callAIForCompression(compressionPrompt)
		if err != nil {
			fmt.Printf("[压缩失败] %v\n", err)
			return
		}
		
		fmt.Printf("[压缩成功] 新摘要长度: %d 字符\n", len(newSummary))
		
		// 更新会话
		session.CompressedSummary = cm.mergeSummaries(session.CompressedSummary, newSummary)
		session.RecentHistory = toKeep
		session.CompressionRound++
		
		// 重要：保存到数据库
		if err := cm.stateManager.SaveSession(session); err != nil {
			fmt.Printf("[压缩保存失败] %v\n", err)
		} else {
			fmt.Printf("[压缩完成] 已保存到数据库，压缩轮次: %d\n", session.CompressionRound)
		}
	}()
}

func (cm *CompressionManager) buildCompressionPrompt(messages []Message) string {
	return fmt.Sprintf(`你是游戏历史记录管理助手。请将以下对话历史压缩为简洁摘要：

🎯 压缩原则：
- 保留重要的游戏进展和状态变化
- 保留关键的角色互动和决策
- 保留影响游戏进程的重要事件
- 保留玩家的重要成就和获得的物品/技能
- 保留未完成的任务和目标

❌ 可以省略：
- 重复的常规操作描述
- 纯粹的环境和氛围描述
- 已解决的临时问题细节
- 无关紧要的过渡性对话

需要压缩的对话：
%s

请用简洁的语言总结以上对话，突出关键的游戏进展（200字以内）：`, 
		cm.formatMessages(messages))
}

func (cm *CompressionManager) formatMessages(messages []Message) string {
	var formatted strings.Builder
	
	for _, msg := range messages {
		formatted.WriteString(fmt.Sprintf("[%s] %s\n", msg.Role, msg.Content))
		
		// 限制显示长度
		if formatted.Len() > 8000 {
			formatted.WriteString("...(内容过长，已截断)")
			break
		}
	}
	
	return formatted.String()
}

func (cm *CompressionManager) callAIForCompression(prompt string) (string, error) {
	// 构建压缩专用的消息
	messages := []services.Message{
		{Role: "user", Content: prompt},
	}
	
	// 获取与游戏一致的AI配置
	if cm.gameController == nil {
		return "", fmt.Errorf("GameController not set for compression")
	}
	
	provider := cm.gameController.defaultProvider
	if provider.APIKey == "" {
		return "", fmt.Errorf("AI provider not configured for compression")
	}
	
	fmt.Printf("[压缩AI调用] 使用配置 - 类型:%s, 模型:%s\n", provider.APIType, provider.ModelID)
	
	// 调用AI进行压缩（使用与游戏相同的配置）
	var response interface{}
	var err error
	
	switch provider.APIType {
	case "openai":
		response, err = cm.aiClient.CallOpenAI(
			provider.BaseURL,
			provider.APIKey,
			provider.ModelID, // 使用与游戏相同的模型
			messages,
			false, // 非流式
		)
	case "anthropic":
		response, err = cm.aiClient.CallAnthropic(
			provider.BaseURL,
			provider.APIKey,
			provider.ModelID,
			messages,
			false,
		)
	case "google":
		response, err = cm.aiClient.CallGoogle(
			provider.BaseURL,
			provider.APIKey,
			provider.ModelID,
			messages,
			false,
		)
	default:
		return "", fmt.Errorf("unsupported API type for compression: %s", provider.APIType)
	}
	
	if err != nil {
		return "", err
	}
	
	if respMap, ok := response.(map[string]interface{}); ok {
		if content, ok := respMap["content"].(string); ok {
			return content, nil
		}
	}
	
	return "", fmt.Errorf("invalid compression response")
}

func (cm *CompressionManager) mergeSummaries(oldSummary, newSummary string) string {
	if oldSummary == "" {
		return newSummary
	}
	
	// 如果已有摘要过长，进行二次压缩
	if len(oldSummary) > 2000 {
		mergePrompt := fmt.Sprintf(`请将以下两个历史摘要合并为一个简洁的总摘要：

旧摘要：%s

新摘要：%s

合并要求：保留最重要的信息，控制在300字以内。`, oldSummary, newSummary)
		
		merged, err := cm.callAIForCompression(mergePrompt)
		if err != nil {
			return oldSummary + "\n" + newSummary // 降级方案
		}
		return merged
	}
	
	return oldSummary + "\n" + newSummary
}