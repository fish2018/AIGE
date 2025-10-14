package models

import (
	"AIGE/config"
)

func AutoMigrate() {
	config.DB.AutoMigrate(&User{}, &Provider{}, &Model{}, &GameSave{}, &SystemConfig{})
}