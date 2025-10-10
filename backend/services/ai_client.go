package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIClient struct {
	client *http.Client
}

func NewAIClient() *AIClient {
	return &AIClient{
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (ai *AIClient) CallOpenAI(baseURL, apiKey, modelID string, messages []Message, stream bool) (interface{}, error) {
	apiURL := ai.buildOpenAIURL(baseURL)

	requestBody := map[string]interface{}{
		"model":       modelID,
		"messages":    messages,
		"temperature": 0.7,
		"max_tokens":  60000,
	}

	if stream {
		requestBody["stream"] = true
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := ai.client.Do(req)
	if err != nil {
		return nil, err
	}

	if !stream {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %d %s", resp.StatusCode, string(body))
	}

	if stream {
		return resp.Body, nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("invalid response format")
	}

	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content := message["content"].(string)

	return map[string]interface{}{
		"content":      content,
		"finishReason": choice["finish_reason"],
	}, nil
}

func (ai *AIClient) CallAnthropic(baseURL, apiKey, modelID string, messages []Message, stream bool) (interface{}, error) {
	apiURL := ai.buildAnthropicURL(baseURL)

	var systemMessage string
	var conversationMessages []Message
	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	anthropicMessages := make([]map[string]interface{}, 0)
	for _, msg := range conversationMessages {
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		anthropicMessages = append(anthropicMessages, map[string]interface{}{
			"role":    role,
			"content": msg.Content,
		})
	}

	requestBody := map[string]interface{}{
		"model":      modelID,
		"max_tokens": 60000,
		"messages":   anthropicMessages,
	}

	if systemMessage != "" {
		requestBody["system"] = systemMessage
	}

	if stream {
		requestBody["stream"] = true
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := ai.client.Do(req)
	if err != nil {
		return nil, err
	}

	if !stream {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error: %d %s", resp.StatusCode, string(body))
	}

	if stream {
		return resp.Body, nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return nil, fmt.Errorf("invalid response format")
	}

	firstContent := content[0].(map[string]interface{})
	text := firstContent["text"].(string)

	return map[string]interface{}{
		"content":      text,
		"finishReason": result["stop_reason"],
	}, nil
}

func (ai *AIClient) CallGoogle(baseURL, apiKey, modelID string, messages []Message, stream bool) (interface{}, error) {
	apiURL := ai.buildGoogleURL(baseURL, modelID, stream)

	var systemMessage string
	var conversationMessages []Message
	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	contents := make([]map[string]interface{}, 0)
	for _, msg := range conversationMessages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		} else {
			role = "user"
		}
		contents = append(contents, map[string]interface{}{
			"role": role,
			"parts": []map[string]interface{}{
				{"text": msg.Content},
			},
		})
	}

	requestBody := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature": 0.7,
		},
	}

	if systemMessage != "" {
		requestBody["system_instruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemMessage},
			},
		}
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := ai.client.Do(req)
	if err != nil {
		return nil, err
	}

	if !stream {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API error: %d %s", resp.StatusCode, string(body))
	}

	if stream {
		return resp.Body, nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return nil, fmt.Errorf("invalid response format")
	}

	candidate := candidates[0].(map[string]interface{})
	content := candidate["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})

	var text string
	for _, part := range parts {
		partMap := part.(map[string]interface{})
		if t, ok := partMap["text"].(string); ok {
			text = t
			break
		}
	}

	return map[string]interface{}{
		"content":      text,
		"finishReason": candidate["finishReason"],
	}, nil
}

func (ai *AIClient) ParseStreamChunk(apiType string, data string) map[string]interface{} {
	switch apiType {
	case "openai":
		return ai.parseOpenAIChunk(data)
	case "anthropic":
		return ai.parseAnthropicChunk(data)
	case "google":
		return ai.parseGoogleChunk(data)
	default:
		return nil
	}
}

func (ai *AIClient) parseOpenAIChunk(data string) map[string]interface{} {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return map[string]interface{}{"content": "", "done": true}
	}

	if strings.HasPrefix(data, "data: ") {
		data = strings.TrimPrefix(data, "data: ")
	}

	if data == "[DONE]" {
		return map[string]interface{}{"content": "", "done": true}
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil
	}

	choices, ok := parsed["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil
	}

	content, _ := delta["content"].(string)
	finishReason, _ := choice["finish_reason"].(string)

	return map[string]interface{}{
		"content": content,
		"done":    finishReason != "",
	}
}

func (ai *AIClient) parseAnthropicChunk(data string) map[string]interface{} {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	if strings.HasPrefix(data, "data: ") {
		data = strings.TrimPrefix(data, "data: ")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil
	}

	eventType, _ := parsed["type"].(string)

	if eventType == "content_block_delta" {
		delta, ok := parsed["delta"].(map[string]interface{})
		if !ok {
			return nil
		}
		content, _ := delta["text"].(string)
		return map[string]interface{}{
			"content": content,
			"done":    false,
		}
	} else if eventType == "message_stop" {
		return map[string]interface{}{
			"content": "",
			"done":    true,
		}
	}

	return nil
}

func (ai *AIClient) parseGoogleChunk(data string) map[string]interface{} {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	if strings.HasPrefix(data, "data: ") {
		data = strings.TrimPrefix(data, "data: ")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil
	}

	candidates, ok := parsed["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return nil
	}

	candidate := candidates[0].(map[string]interface{})
	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return nil
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return nil
	}

	var text string
	for _, part := range parts {
		partMap := part.(map[string]interface{})
		if t, ok := partMap["text"].(string); ok {
			text = t
			break
		}
	}

	finishReason, _ := candidate["finishReason"].(string)

	return map[string]interface{}{
		"content": text,
		"done":    finishReason != "",
	}
}

func (ai *AIClient) buildOpenAIURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if strings.Contains(baseURL, "/chat/completions") {
		return baseURL
	}
	if strings.Contains(baseURL, "/v1") {
		return strings.TrimSuffix(baseURL, "/") + "/chat/completions"
	}
	return strings.TrimSuffix(baseURL, "/") + "/v1/chat/completions"
}

func (ai *AIClient) buildAnthropicURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if strings.Contains(baseURL, "/v1/messages") {
		return baseURL
	}
	return strings.TrimSuffix(baseURL, "/") + "/v1/messages"
}

func (ai *AIClient) buildGoogleURL(baseURL, modelID string, stream bool) string {
	baseURL = strings.TrimSpace(baseURL)

	if strings.Contains(baseURL, "/models/") {
		baseURL = strings.Split(baseURL, "/models/")[0]
	}

	if !strings.HasSuffix(baseURL, "/v1beta") {
		baseURL = strings.TrimSuffix(baseURL, "/") + "/v1beta"
	}

	endpoint := "generateContent"
	if stream {
		endpoint = "streamGenerateContent"
		// 关键：添加alt=sse参数以获取SSE格式流
		return fmt.Sprintf("%s/models/%s:%s?alt=sse", baseURL, modelID, endpoint)
	}

	return fmt.Sprintf("%s/models/%s:%s", baseURL, modelID, endpoint)
}

func ReadStreamLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
