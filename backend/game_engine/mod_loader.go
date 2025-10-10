package game_engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ModConfig represents the configuration of a game mod
type ModConfig struct {
	GameID      string `json:"game_id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`

	GameConfig struct {
		InitialOpportunities  int     `json:"initial_opportunities"`
		RewardScalingFactor   int     `json:"reward_scaling_factor"`
		MaxTokenHistory       int     `json:"max_token_history"`
		AutoSaveInterval      int     `json:"auto_save_interval"`
		RollSettings          struct {
			CriticalSuccessThreshold float64 `json:"critical_success_threshold"`
			CriticalFailureThreshold float64 `json:"critical_failure_threshold"`
			DefaultSides             int     `json:"default_sides"`
		} `json:"roll_settings"`
		CheatCheck struct {
			Enabled       bool   `json:"enabled"`
			CheckInterval int    `json:"check_interval"`
			Model         string `json:"model"`
		} `json:"cheat_check"`
	} `json:"game_config"`

	Prompts map[string]string `json:"prompts"`

	InitialState  map[string]interface{} `json:"initial_state"`
	WelcomeMessage string                 `json:"welcome_message"`
}

// GameMod represents a loaded game mod
type GameMod struct {
	Config    ModConfig
	ModPath   string
	Prompts   map[string]string // prompt name -> prompt content
}

// ModLoader handles loading and managing game mods
type ModLoader struct {
	ModsPath string
	LoadedMods map[string]*GameMod
}

// NewModLoader creates a new mod loader
func NewModLoader(modsPath string) *ModLoader {
	return &ModLoader{
		ModsPath:   modsPath,
		LoadedMods: make(map[string]*GameMod),
	}
}

// LoadMod loads a specific game mod
func (ml *ModLoader) LoadMod(modID string) (*GameMod, error) {
	// Check if already loaded
	if mod, exists := ml.LoadedMods[modID]; exists {
		return mod, nil
	}

	modPath := filepath.Join(ml.ModsPath, modID)
	
	// Check if mod directory exists
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("mod '%s' not found in %s", modID, ml.ModsPath)
	}

	// Load config.json
	configPath := filepath.Join(modPath, "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mod config: %w", err)
	}

	var config ModConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse mod config: %w", err)
	}

	// Load prompts
	prompts := make(map[string]string)
	for promptName, promptPath := range config.Prompts {
		fullPath := filepath.Join(modPath, promptPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load prompt '%s': %w", promptName, err)
		}
		prompts[promptName] = string(content)
	}

	mod := &GameMod{
		Config:  config,
		ModPath: modPath,
		Prompts: prompts,
	}

	ml.LoadedMods[modID] = mod
	return mod, nil
}

// GetMod retrieves a loaded mod
func (ml *ModLoader) GetMod(modID string) (*GameMod, error) {
	if mod, exists := ml.LoadedMods[modID]; exists {
		return mod, nil
	}
	return nil, fmt.Errorf("mod '%s' not loaded", modID)
}

// ListAvailableMods lists all available mods in the mods directory
func (ml *ModLoader) ListAvailableMods() ([]string, error) {
	entries, err := os.ReadDir(ml.ModsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mods directory: %w", err)
	}

	var mods []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it's a valid mod (has config.json)
			configPath := filepath.Join(ml.ModsPath, entry.Name(), "config.json")
			if _, err := os.Stat(configPath); err == nil {
				mods = append(mods, entry.Name())
			}
		}
	}

	return mods, nil
}

// ReloadMod reloads a specific mod
func (ml *ModLoader) ReloadMod(modID string) error {
	delete(ml.LoadedMods, modID)
	_, err := ml.LoadMod(modID)
	return err
}

// LoadMods loads all available mods from the mods directory
func (ml *ModLoader) LoadMods(modsPath string) error {
	ml.ModsPath = modsPath
	
	availableMods, err := ml.ListAvailableMods()
	if err != nil {
		return fmt.Errorf("failed to list mods: %w", err)
	}

	fmt.Printf("找到 %d 个可用mod\n", len(availableMods))
	for _, modID := range availableMods {
		if _, err := ml.LoadMod(modID); err != nil {
			fmt.Printf("警告: 加载mod '%s' 失败: %v\n", modID, err)
			continue
		}
		fmt.Printf("成功加载mod: %s\n", modID)
	}

	return nil
}

// GetAllMods returns all loaded mods
func (ml *ModLoader) GetAllMods() []*GameMod {
	mods := make([]*GameMod, 0, len(ml.LoadedMods))
	for _, mod := range ml.LoadedMods {
		mods = append(mods, mod)
	}
	return mods
}
