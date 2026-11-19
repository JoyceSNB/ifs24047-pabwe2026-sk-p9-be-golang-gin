package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"app/internal/database"
)

type AiClientInterface interface {
	CheckAccess() error
	ChatCompletion(topic, category string, searchSnippets []string) (*GeneratedPost, error)
}

type openAiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAiRequest struct {
	Model       string          `json:"model"`
	Messages    []openAiMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type openAiChoice struct {
	Message openAiMessage `json:"message"`
}

type openAiResponse struct {
	Choices []openAiChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type OpenAiClient struct {
	Endpoint   string
	Key        string
	Model      string
	HttpClient *http.Client
}

func NewOpenAiClient(endpoint, key, model string) *OpenAiClient {
	return &OpenAiClient{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Key:      key,
		Model:    model,
		HttpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *OpenAiClient) CheckAccess() error {
	if strings.TrimSpace(c.Key) == "" {
		return errors.New("OpenAI API key is not configured")
	}
	url := fmt.Sprintf("%s/models", c.Endpoint)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Key))

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(respBytes))
	}
	return nil
}

func (c *OpenAiClient) ChatCompletion(topic, category string, searchSnippets []string) (*GeneratedPost, error) {
	systemInstruction := fmt.Sprintf(`Kamu adalah penulis blog profesional berbahasa Indonesia.
Tulis SATU artikel blog lengkap dalam format JSON murni TANPA markdown block, TANPA kata pembuka/penutup.
Format JSON wajib:
{"title": "Judul artikel", "category": "salah satu dari: %s", "excerpt": "ringkasan 1-2 kalimat maksimal 160 karakter", "content": "isi artikel dalam format Markdown (gunakan heading ##, paragraf, dan list), panjang sekitar 500 sampai 700 kata", "sources": ["URL rujukan jika ada"]}`, strings.Join(database.Categories, ", "))

	var userPromptBuilder strings.Builder
	userPromptBuilder.WriteString(fmt.Sprintf("Topik artikel: %s", topic))
	if strings.TrimSpace(category) != "" {
		userPromptBuilder.WriteString(fmt.Sprintf("\nKategori yang diinginkan: %s", category))
	}

	if len(searchSnippets) > 0 {
		userPromptBuilder.WriteString("\n\nKonteks Penelusuran Mesin Pencari / Search Engine:\n")
		for i, snippet := range searchSnippets {
			userPromptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, snippet))
		}
		userPromptBuilder.WriteString("\nGunakan fakta dan rujukan di atas untuk menulis artikel yang akurat dan sertakan tautan rujukannya pada field sources.")
	}

	reqPayload := openAiRequest{
		Model: c.Model,
		Messages: []openAiMessage{
			{Role: "system", Content: systemInstruction},
			{Role: "user", Content: userPromptBuilder.String()},
		},
		Temperature: 0.7,
	}

	bodyBytes, _ := json.Marshal(reqPayload)
	url := fmt.Sprintf("%s/chat/completions", c.Endpoint)

	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Key))

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var aiResp openAiResponse
	if err := json.Unmarshal(respBytes, &aiResp); err != nil {
		return nil, err
	}

	if len(aiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from OpenAI API")
	}

	rawContent := strings.TrimSpace(aiResp.Choices[0].Message.Content)
	// Ambil objek JSON di antara kurung kurawal pertama dan terakhir. Ini menangani model
	// yang membungkus jawaban dengan ```json ... ``` atau menambahkan teks pembuka/penutup.
	if start, end := strings.Index(rawContent, "{"), strings.LastIndex(rawContent, "}"); start >= 0 && end > start {
		rawContent = rawContent[start : end+1]
	}

	var post GeneratedPost
	if err := json.Unmarshal([]byte(rawContent), &post); err != nil {
		return nil, fmt.Errorf("failed to parse AI response into blog post: %w", err)
	}

	return &post, nil
}