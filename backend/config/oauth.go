package config

import (
	"encoding/json"
)

type OAuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"user_info_url"`
	Enabled      bool   `json:"enabled"`
}

func GetLinuxDoOAuthConfig() (*OAuthConfig, error) {
	var config OAuthConfig
	
	var clientID, clientSecret, redirectURL, authURL, tokenURL, userInfoURL, enabled string
	
	err := DB.Model(&SystemConfig{}).Where("key = ?", "oauth_linux_do_client_id").Select("value").Scan(&clientID).Error
	if err == nil && clientID != "" {
		config.ClientID = clientID
	}
	
	err = DB.Model(&SystemConfig{}).Where("key = ?", "oauth_linux_do_client_secret").Select("value").Scan(&clientSecret).Error
	if err == nil && clientSecret != "" {
		config.ClientSecret = clientSecret
	}
	
	err = DB.Model(&SystemConfig{}).Where("key = ?", "oauth_linux_do_redirect_url").Select("value").Scan(&redirectURL).Error
	if err == nil && redirectURL != "" {
		config.RedirectURL = redirectURL
	}
	
	err = DB.Model(&SystemConfig{}).Where("key = ?", "oauth_linux_do_auth_url").Select("value").Scan(&authURL).Error
	if err == nil && authURL != "" {
		config.AuthURL = authURL
	} else {
		config.AuthURL = "https://connect.linux.do/oauth2/authorize"
	}
	
	err = DB.Model(&SystemConfig{}).Where("key = ?", "oauth_linux_do_token_url").Select("value").Scan(&tokenURL).Error
	if err == nil && tokenURL != "" {
		config.TokenURL = tokenURL
	} else {
		config.TokenURL = "https://connect.linux.do/oauth2/token"
	}
	
	err = DB.Model(&SystemConfig{}).Where("key = ?", "oauth_linux_do_user_info_url").Select("value").Scan(&userInfoURL).Error
	if err == nil && userInfoURL != "" {
		config.UserInfoURL = userInfoURL
	} else {
		config.UserInfoURL = "https://connect.linux.do/api/user"
	}
	
	err = DB.Model(&SystemConfig{}).Where("key = ?", "oauth_linux_do_enabled").Select("value").Scan(&enabled).Error
	if err == nil && enabled == "true" {
		config.Enabled = true
	} else {
		config.Enabled = false
	}
	
	return &config, nil
}

func SaveLinuxDoOAuthConfig(config *OAuthConfig) error {
	configs := map[string]string{
		"oauth_linux_do_client_id":      config.ClientID,
		"oauth_linux_do_client_secret":  config.ClientSecret,
		"oauth_linux_do_redirect_url":   config.RedirectURL,
		"oauth_linux_do_auth_url":       config.AuthURL,
		"oauth_linux_do_token_url":      config.TokenURL,
		"oauth_linux_do_user_info_url":  config.UserInfoURL,
		"oauth_linux_do_enabled":        jsonBool(config.Enabled),
	}
	
	for key, value := range configs {
		var sc SystemConfig
		err := DB.Where("key = ?", key).First(&sc).Error
		if err != nil {
			sc = SystemConfig{Key: key, Value: value}
			if err := DB.Create(&sc).Error; err != nil {
				return err
			}
		} else {
			sc.Value = value
			if err := DB.Save(&sc).Error; err != nil {
				return err
			}
		}
	}
	
	return nil
}

func jsonBool(b bool) string {
	data, _ := json.Marshal(b)
	return string(data)
}

type SystemConfig struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Key       string `json:"key" gorm:"uniqueIndex;not null"`
	Value     string `json:"value" gorm:"type:text"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
