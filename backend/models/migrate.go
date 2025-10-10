package models

import (
	"AIGE/config"
)

func AutoMigrate() {
	config.DB.AutoMigrate(&User{}, &ChatMessage{}, &Provider{}, &Model{}, &GameSave{}, &SystemConfig{})
}