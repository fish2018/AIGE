package controllers

import (
	"AIGE/config"
	"AIGE/models"
	"AIGE/utils"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type LinuxDoUserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar_url"`
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func LinuxDoLogin(c *gin.Context) {
	oauthConfig, err := config.GetLinuxDoOAuthConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get OAuth config"})
		return
	}

	if !oauthConfig.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "Linux.Do OAuth is not enabled"})
		return
	}

	if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OAuth not configured"})
		return
	}

	state := generateState()
	
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=read",
		oauthConfig.AuthURL,
		url.QueryEscape(oauthConfig.ClientID),
		url.QueryEscape(oauthConfig.RedirectURL),
		state,
	)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

func LinuxDoCallback(c *gin.Context) {
	oauthConfig, err := config.GetLinuxDoOAuthConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get OAuth config"})
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	cookieState, err := c.Cookie("oauth_state")
	if err != nil || state != cookieState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	tokenData := url.Values{}
	tokenData.Set("grant_type", "authorization_code")
	tokenData.Set("code", code)
	tokenData.Set("redirect_uri", oauthConfig.RedirectURL)
	tokenData.Set("client_id", oauthConfig.ClientID)
	tokenData.Set("client_secret", oauthConfig.ClientSecret)

	req, err := http.NewRequest("POST", oauthConfig.TokenURL, strings.NewReader(tokenData.Encode()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token request"})
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get access token",
			"details": string(body),
		})
		return
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token response"})
		return
	}

	userReq, err := http.NewRequest("GET", oauthConfig.UserInfoURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user info request"})
		return
	}

	userReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenResponse.AccessToken))
	userReq.Header.Set("Accept", "application/json")

	userResp, err := client.Do(userReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer userResp.Body.Close()

	userBody, _ := io.ReadAll(userResp.Body)

	if userResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user info",
			"details": string(userBody),
		})
		return
	}

	var userInfo LinuxDoUserInfo
	if err := json.Unmarshal(userBody, &userInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
		return
	}

	var user models.User
	oauthID := fmt.Sprintf("%d", userInfo.ID)
	
	result := config.DB.Where("oauth_provider = ? AND oauth_id = ?", "linux-do", oauthID).First(&user)
	
	if result.Error != nil {
		email := userInfo.Email
		if email == "" {
			email = fmt.Sprintf("linux-do-%d@oauth.local", userInfo.ID)
		}

		var existingEmailUser models.User
		if config.DB.Where("email = ?", email).First(&existingEmailUser).Error == nil {
			existingEmailUser.OAuthProvider = "linux-do"
			existingEmailUser.OAuthID = oauthID
			if userInfo.Avatar != "" {
				existingEmailUser.Avatar = userInfo.Avatar
			}
			config.DB.Save(&existingEmailUser)
			user = existingEmailUser
		} else {
			username := userInfo.Username
			var existingUser models.User
			for i := 0; ; i++ {
				checkUsername := username
				if i > 0 {
					checkUsername = fmt.Sprintf("%s_%d", username, i)
				}
				
				if err := config.DB.Where("username = ?", checkUsername).First(&existingUser).Error; err != nil {
					username = checkUsername
					break
				}
			}

			user = models.User{
				Username:      username,
				Email:         email,
				OAuthProvider: "linux-do",
				OAuthID:       oauthID,
				Avatar:        userInfo.Avatar,
				IsAdmin:       false,
			}

			if err := config.DB.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "创建用户失败", 
					"details": err.Error(),
				})
				return
			}
		}
	} else {
		if userInfo.Avatar != "" {
			user.Avatar = userInfo.Avatar
		}
		if userInfo.Username != "" {
			user.Username = userInfo.Username
		}
		if userInfo.Email != "" && user.Email != userInfo.Email {
			var emailCheck models.User
			if err := config.DB.Where("email = ? AND id != ?", userInfo.Email, user.ID).First(&emailCheck).Error; err != nil {
				user.Email = userInfo.Email
			}
		}
		config.DB.Save(&user)
	}

	token, err := utils.GenerateJWT(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
	})
}


func GetOAuthConfig(c *gin.Context) {
	oauthConfig, err := config.GetLinuxDoOAuthConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get OAuth config"})
		return
	}

	c.JSON(http.StatusOK, oauthConfig)
}

func SaveOAuthConfig(c *gin.Context) {
	var oauthConfig config.OAuthConfig
	if err := c.ShouldBindJSON(&oauthConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := config.SaveLinuxDoOAuthConfig(&oauthConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save OAuth config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OAuth config saved successfully"})
}
