package game_engine

import (
	"AIGE/config"
	"AIGE/models"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// GameSession represents a player's game session
type GameSession struct {
	PlayerID         string                 `json:"player_id"`
	ModID            string                 `json:"mod_id"`
	SessionDate      string                 `json:"session_date"`
	State            map[string]interface{} `json:"state"`        // Dynamic state based on mod config
	RecentHistory    []Message              `json:"recent_history"`     // 最近4条对话
	CompressedSummary string                `json:"compressed_summary"` // 压缩摘要
	CompressionRound int                    `json:"compression_round"`  // 压缩轮次
	DisplayHistory   []string               `json:"display_history"`  // User-facing narrative
	LastModified     time.Time              `json:"last_modified"`
	
	// 预留社交功能字段
	Social *SocialData `json:"social,omitempty"` // 社交数据（预留）
}

// SocialData 社交相关数据（预留扩展）
type SocialData struct {
	Friends      []string               `json:"friends,omitempty"`       // 好友列表
	Team         *TeamInfo              `json:"team,omitempty"`          // 队伍信息
	TradeOffers  []TradeOffer           `json:"trade_offers,omitempty"`  // 交易请求
	ChatRooms    []string               `json:"chat_rooms,omitempty"`    // 加入的聊天室
	Reputation   int                    `json:"reputation,omitempty"`    // 声望值
	Guild        string                 `json:"guild,omitempty"`         // 公会/宗门
}

// TeamInfo 队伍信息（预留）
type TeamInfo struct {
	TeamID    string   `json:"team_id"`
	Leader    string   `json:"leader"`
	Members   []string `json:"members"`
	Quest     string   `json:"quest,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TradeOffer 交易请求（预留）
type TradeOffer struct {
	OfferID    string                 `json:"offer_id"`
	FromPlayer string                 `json:"from_player"`
	ToPlayer   string                 `json:"to_player"`
	Items      map[string]interface{} `json:"items"`
	Status     string                 `json:"status"` // pending, accepted, rejected
	CreatedAt  time.Time             `json:"created_at"`
}

// Message represents a chat message in the AI conversation
type Message struct {
	Role      string    `json:"role"`    // system, user, assistant
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// StateManager handles game session storage and retrieval
type StateManager struct {
	mu          sync.RWMutex
	sessions    map[string]map[string]*GameSession // playerID -> modID -> session
	autoSave    bool
	saveInterval time.Duration
}

// NewStateManager creates a new state manager
func NewStateManager(autoSave bool, saveInterval time.Duration) *StateManager {
	sm := &StateManager{
		sessions:     make(map[string]map[string]*GameSession),
		autoSave:     autoSave,
		saveInterval: saveInterval,
	}
	
	if autoSave {
		go sm.autoSaveLoop()
	}
	
	return sm
}

// GetSession retrieves a player's session for a specific mod
func (sm *StateManager) GetSession(playerID, modID string) (*GameSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	playerSessions, exists := sm.sessions[playerID]
	if !exists {
		session, err := sm.loadFromDB(playerID, modID)
		if err != nil {
			return nil, fmt.Errorf("no sessions found for player %s", playerID)
		}
		if sm.sessions[playerID] == nil {
			sm.sessions[playerID] = make(map[string]*GameSession)
		}
		sm.sessions[playerID][modID] = session
		sm.checkAndResetDaily(session)
		return session, nil
	}
	
	session, exists := playerSessions[modID]
	if !exists {
		session, err := sm.loadFromDB(playerID, modID)
		if err != nil {
			return nil, fmt.Errorf("session not found for player %s in mod %s", playerID, modID)
		}
		sm.sessions[playerID][modID] = session
		sm.checkAndResetDaily(session)
		return session, nil
	}
	
	sm.checkAndResetDaily(session)
	
	return session, nil
}

// checkAndResetDaily 检查并执行每日重置
func (sm *StateManager) checkAndResetDaily(session *GameSession) {
	today := time.Now().Format("2006-01-02")
	
	// 如果session_date与今天不同，说明是新的一天，需要重置
	if session.SessionDate != today {
		fmt.Printf("[StateManager] 检测到新的一天，重置玩家 %s 的机缘\n", session.PlayerID)
		
		// 重置机缘数
		if session.State != nil {
			session.State["opportunities_remaining"] = 10
			session.State["daily_success_achieved"] = false
			session.State["is_in_trial"] = false
			session.State["is_processing"] = false
			session.State["current_life"] = nil
		}
		
		// 更新日期
		session.SessionDate = today
		session.LastModified = time.Now()
	}
}

// GetPlayerSessions retrieves all sessions for a player
func (sm *StateManager) GetPlayerSessions(playerID string) (map[string]*GameSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	sessions, exists := sm.sessions[playerID]
	if !exists {
		return make(map[string]*GameSession), nil // Return empty map, not an error
	}
	
	// Return a copy to avoid external modifications
	sessionsCopy := make(map[string]*GameSession)
	for modID, session := range sessions {
		sessionsCopy[modID] = session
	}
	
	return sessionsCopy, nil
}

// SaveSession saves or updates a player's session
func (sm *StateManager) SaveSession(session *GameSession) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session.LastModified = time.Now()
	
	if sm.sessions[session.PlayerID] == nil {
		sm.sessions[session.PlayerID] = make(map[string]*GameSession)
	}
	
	sm.sessions[session.PlayerID][session.ModID] = session
	
	return sm.saveToDB(session)
}

// CreateSession creates a new game session
func (sm *StateManager) CreateSession(playerID, modID string, initialState map[string]interface{}, systemPrompt string) (*GameSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session := &GameSession{
		PlayerID:         playerID,
		ModID:            modID,
		SessionDate:      time.Now().Format("2006-01-02"),
		State:            initialState,
		RecentHistory:    []Message{}, // 不再存储系统提示词
		CompressedSummary: "",
		CompressionRound: 0,
		DisplayHistory:   []string{},
		LastModified:     time.Now(),
	}
	
	// Ensure player's sessions map exists
	if sm.sessions[playerID] == nil {
		sm.sessions[playerID] = make(map[string]*GameSession)
	}
	
	sm.sessions[playerID][modID] = session
	return session, nil
}

// DeleteSession removes a specific player's session for a mod
func (sm *StateManager) DeleteSession(playerID, modID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Delete from memory
	if playerSessions, exists := sm.sessions[playerID]; exists {
		delete(playerSessions, modID)
		// If player has no more sessions, remove player entry
		if len(playerSessions) == 0 {
			delete(sm.sessions, playerID)
		}
	}
	
	// Delete from database (physical deletion)
	userID, err := strconv.ParseUint(playerID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid player ID: %w", err)
	}
	
	result := config.DB.Unscoped().Where("user_id = ? AND mod_id = ?", userID, modID).Delete(&models.GameSave{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session from database: %w", result.Error)
	}
	
	fmt.Printf("[StateManager] 物理删除存档: 用户%s mod%s, 删除了%d条记录\n", playerID, modID, result.RowsAffected)
	
	return nil
}

// DeletePlayerSessions removes all sessions for a player
func (sm *StateManager) DeletePlayerSessions(playerID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Delete from memory
	delete(sm.sessions, playerID)
	
	// Delete from database (physical deletion)
	userID, err := strconv.ParseUint(playerID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid player ID: %w", err)
	}
	
	result := config.DB.Unscoped().Where("user_id = ?", userID).Delete(&models.GameSave{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete player sessions from database: %w", result.Error)
	}
	
	fmt.Printf("[StateManager] 物理删除玩家所有存档: 用户%s, 删除了%d条记录\n", playerID, result.RowsAffected)
	
	return nil
}

// loadFromDB loads a session from database
func (sm *StateManager) loadFromDB(playerID, modID string) (*GameSession, error) {
	userID, err := strconv.ParseUint(playerID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid player ID: %w", err)
	}
	
	var gameSave models.GameSave
	result := config.DB.Where("user_id = ? AND mod_id = ?", userID, modID).First(&gameSave)
	if result.Error != nil {
		return nil, result.Error
	}
	
	var state map[string]interface{}
	if err := json.Unmarshal([]byte(gameSave.State), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	
	var recentHistory []Message
	if gameSave.RecentHistory != "" {
		if err := json.Unmarshal([]byte(gameSave.RecentHistory), &recentHistory); err != nil {
			return nil, fmt.Errorf("failed to unmarshal recent history: %w", err)
		}
	}
	
	var displayHistory []string
	if gameSave.DisplayHistory != "" {
		if err := json.Unmarshal([]byte(gameSave.DisplayHistory), &displayHistory); err != nil {
			return nil, fmt.Errorf("failed to unmarshal display history: %w", err)
		}
	}
	
	session := &GameSession{
		PlayerID:         playerID,
		ModID:            modID,
		SessionDate:      gameSave.SessionDate,
		State:            state,
		RecentHistory:    recentHistory,
		CompressedSummary: gameSave.CompressedSummary,
		CompressionRound: gameSave.CompressionRound,
		DisplayHistory:   displayHistory,
		LastModified:     gameSave.UpdatedAt,
	}
	
	return session, nil
}

// saveToDB saves a session to database
func (sm *StateManager) saveToDB(session *GameSession) error {
	userID, err := strconv.ParseUint(session.PlayerID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid player ID: %w", err)
	}
	
	stateJSON, err := json.Marshal(session.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	
	recentHistoryJSON, err := json.Marshal(session.RecentHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal recent history: %w", err)
	}
	
	displayHistoryJSON, err := json.Marshal(session.DisplayHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal display history: %w", err)
	}
	
	gameSave := models.GameSave{
		UserID:           uint(userID),
		ModID:            session.ModID,
		SessionDate:      session.SessionDate,
		State:            string(stateJSON),
		RecentHistory:    string(recentHistoryJSON),
		CompressedSummary: session.CompressedSummary,
		CompressionRound: session.CompressionRound,
		DisplayHistory:   string(displayHistoryJSON),
	}
	
	// Use ON CONFLICT (upsert) to ensure only one record per user_id + mod_id
	result := config.DB.Where("user_id = ? AND mod_id = ?", userID, session.ModID).
		Assign(gameSave).
		FirstOrCreate(&gameSave)
	
	return result.Error
}

// SaveToFile saves all sessions to persistent storage (deprecated, kept for compatibility)
func (sm *StateManager) SaveToFile() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	for playerID, playerSessions := range sm.sessions {
		for _, session := range playerSessions {
			if err := sm.saveToDB(session); err != nil {
				fmt.Printf("Failed to save session for player %s mod %s: %v\n", playerID, session.ModID, err)
			}
		}
	}
	
	return nil
}

// autoSaveLoop periodically saves sessions to file
func (sm *StateManager) autoSaveLoop() {
	ticker := time.NewTicker(sm.saveInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		if err := sm.SaveToFile(); err != nil {
			fmt.Printf("Auto-save error: %v\n", err)
		}
	}
}

// ApplyStateUpdate applies a state update using dot notation
// Example: "current_life.hp": 100
func ApplyStateUpdate(state map[string]interface{}, updates map[string]interface{}) error {
	for key, value := range updates {
		if err := setNestedValue(state, key, value); err != nil {
			return err
		}
	}
	return nil
}

// setNestedValue sets a value in a nested map using dot notation
func setNestedValue(m map[string]interface{}, path string, value interface{}) error {
	keys := splitPath(path)
	
	if len(keys) == 0 {
		return fmt.Errorf("empty path")
	}
	
	// Navigate to the parent of the target key
	current := m
	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			// Create intermediate maps as needed (handle null values)
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}
	
	// Handle array append operation (key ends with "+")
	finalKey := keys[len(keys)-1]
	if len(finalKey) > 0 && finalKey[len(finalKey)-1] == '+' {
		actualKey := finalKey[:len(finalKey)-1]
		
		// Get or create the array
		var arr []interface{}
		if existing, ok := current[actualKey]; ok {
			if existingArr, ok := existing.([]interface{}); ok {
				arr = existingArr
			}
		}
		
		// Append the value
		if valueArr, ok := value.([]interface{}); ok {
			arr = append(arr, valueArr...)
		} else {
			arr = append(arr, value)
		}
		
		current[actualKey] = arr
	} else {
		// Regular assignment
		current[finalKey] = value
	}
	
	return nil
}

// splitPath splits a dot-notation path into keys
func splitPath(path string) []string {
	var keys []string
	var current string
	
	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				keys = append(keys, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	
	if current != "" {
		keys = append(keys, current)
	}
	
	return keys
}

// GetAllSessions returns all active sessions (for admin or live viewing)
func (sm *StateManager) GetAllSessions() map[string]map[string]*GameSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Return a copy to avoid external modifications
	allSessions := make(map[string]map[string]*GameSession)
	for playerID, playerSessions := range sm.sessions {
		allSessions[playerID] = make(map[string]*GameSession)
		for modID, session := range playerSessions {
			allSessions[playerID][modID] = session
		}
	}
	
	return allSessions
}

// GetAllSessionsForMod returns all active sessions for a specific mod (for live viewing)
func (sm *StateManager) GetAllSessionsForMod(modID string) map[string]*GameSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	modSessions := make(map[string]*GameSession)
	for playerID, playerSessions := range sm.sessions {
		if session, exists := playerSessions[modID]; exists {
			modSessions[playerID] = session
		}
	}
	
	return modSessions
}
