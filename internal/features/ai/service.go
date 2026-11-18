package ai

import (
	"errors"
	"fmt"
	"strings"

	"app/internal/database"
)

type AiService interface {
	GenerateBlog(req GenerateBlogRequest) (*GeneratedPost, error)
}

type aiService struct {
	client       AiClientInterface
	searchClient SearchEngineClient
}

func NewAiService(client AiClientInterface, searchClient SearchEngineClient) AiService {
	return &aiService{
		client:       client,
		searchClient: searchClient,
	}
}

func (s *aiService) GenerateBlog(req GenerateBlogRequest) (*GeneratedPost, error) {
	if strings.TrimSpace(req.Topic) == "" {
		return nil, errors.New("topic is required")
	}

	// 1. Cek koneksi OpenAI terlebih dahulu
	if err := s.client.CheckAccess(); err != nil {
		return nil, fmt.Errorf("gagal mengakses OpenAI: %w", err)
	}

	// 2. Jika dicentang, cari fakta terkini di Search Engine
	var searchSnippets []string
	if req.IncludeWebSearch && s.searchClient != nil {
		snippets, err := s.searchClient.Search(req.Topic)
		if err == nil && len(snippets) > 0 {
			searchSnippets = snippets
		}
	}

	// 3. Barulah tulis draf artikel
	post, err := s.client.ChatCompletion(req.Topic, req.Category, searchSnippets)
	if err != nil {
		return nil, err
	}

	// 4. Pastikan kategori valid; jika tidak, pakai kategori permintaan atau "Umum"
	category, ok := database.NormalizeCategory(post.Category)
	if !ok {
		category, ok = database.NormalizeCategory(req.Category)
		if !ok {
			category = "Umum"
		}
	}
	post.Category = category

	// 5. Jika model tidak menyertakan sumber, pakai hasil penelusuran sebagai rujukan
	if len(post.Sources) == 0 && len(searchSnippets) > 0 {
		post.Sources = searchSnippets
	}

	return post, nil
}
