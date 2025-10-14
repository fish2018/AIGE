package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"not null"`
	Email     string         `json:"email" gorm:"uniqueIndex"`
	IsAdmin   bool           `json:"is_admin" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}


type Provider struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"not null"`
	Type          string         `json:"type" gorm:"not null;index"`
	APIKey        string         `json:"api_key" gorm:"not null"`
	BaseURL       string         `json:"base_url"`
	Enabled       bool           `json:"enabled" gorm:"default:true"`
	AllowCustomURL bool          `json:"allow_custom_url" gorm:"default:true"`
	Models        []Model        `json:"models" gorm:"foreignKey:ProviderID"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type Model struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	ModelID      string         `json:"model_id" gorm:"not null;index"`
	Name         string         `json:"name" gorm:"not null"`
	ProviderID   uint           `json:"provider_id" gorm:"not null;index"`
	Provider     Provider       `json:"provider" gorm:"foreignKey:ProviderID"`
	Enabled      bool           `json:"enabled" gorm:"default:true"`
	APIType      string         `json:"api_type"`
	Capabilities string         `json:"capabilities" gorm:"type:text"`
	LastTested   *time.Time     `json:"last_tested"`
	TestStatus   string         `json:"test_status" gorm:"default:'untested'"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

type GameSave struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"user_id" gorm:"not null;uniqueIndex:idx_user_mod"`
	ModID            string         `json:"mod_id" gorm:"not null;uniqueIndex:idx_user_mod"`
	SessionDate      string         `json:"session_date" gorm:"not null"`
	State            string         `json:"state" gorm:"type:text;not null"`
	RecentHistory    string         `json:"recent_history" gorm:"type:text"`
	CompressedSummary string        `json:"compressed_summary" gorm:"type:text"`
	CompressionRound int            `json:"compression_round" gorm:"default:0"`
	DisplayHistory   string         `json:"display_history" gorm:"type:text"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

type SystemConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}