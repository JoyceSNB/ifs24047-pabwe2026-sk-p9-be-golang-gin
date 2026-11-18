package post

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"app/internal/database"
)

var (
	ErrPostNotFound    = errors.New("post not found")
	ErrForbidden       = errors.New("forbidden: you do not own this post")
	ErrInvalidCategory = errors.New("invalid category")
	ErrUserNotFound    = errors.New("user not found")
)

type PostService interface {
	// Penulis
	GetPosts(userID uint, q, category string) ([]database.Post, error)
	GetPostByID(userID uint, id uint) (*database.Post, error)
	CreatePost(userID uint, req CreatePostRequest) (*database.Post, error)
	UpdatePost(userID uint, id uint, req UpdatePostRequest) (*database.Post, error)
	TogglePublish(userID uint, id uint) (*database.Post, error)
	DeletePost(userID uint, id uint) error
	GetStats(userID uint) (*PostStatsResponse, error)

	// Publik
	GetCategories() []string
	GetPublicPosts(q, category string) ([]database.Post, error)
	GetPublicPostBySlug(slug string) (*database.Post, error)

	// Admin
	GetAllUsers() ([]UserWithStats, error)
	GetAllPosts() ([]database.Post, error)
	GetUserPosts(targetUserID uint) ([]database.Post, error)
}

type postService struct {
	repo PostRepository
}

func NewPostService(repo PostRepository) PostService {
	return &postService{repo: repo}
}

// normalizeCategory memvalidasi kategori; kategori kosong menjadi "Umum".
func normalizeCategory(category string) (string, error) {
	if strings.TrimSpace(category) == "" {
		return "Umum", nil
	}
	canonical, ok := database.NormalizeCategory(category)
	if !ok {
		return "", ErrInvalidCategory
	}
	return canonical, nil
}

// makeExcerpt membuat ringkasan maksimal 160 karakter dari isi artikel.
func makeExcerpt(content string) string {
	text := strings.Join(strings.Fields(content), " ")
	runes := []rune(text)
	if len(runes) <= 160 {
		return text
	}
	return string(runes[:160]) + "..."
}

// uniqueSlug membuat slug dari judul dan menambahkan akhiran angka jika sudah dipakai.
func (s *postService) uniqueSlug(title string, excludeID uint) string {
	base := database.Slugify(title)
	slug := base
	for i := 2; s.repo.SlugExists(slug, excludeID); i++ {
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return slug
}

func (s *postService) GetPosts(userID uint, q, category string) ([]database.Post, error) {
	return s.repo.FindByUserID(userID, q, category)
}

func (s *postService) GetPostByID(userID uint, id uint) (*database.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if post.UserID != userID {
		return nil, ErrForbidden
	}
	return post, nil
}

func (s *postService) CreatePost(userID uint, req CreatePostRequest) (*database.Post, error) {
	category, err := normalizeCategory(req.Category)
	if err != nil {
		return nil, err
	}

	excerpt := strings.TrimSpace(req.Excerpt)
	if excerpt == "" {
		excerpt = makeExcerpt(req.Content)
	}

	post := &database.Post{
		UserID:         userID,
		Title:          req.Title,
		Slug:           s.uniqueSlug(req.Title, 0),
		Category:       category,
		Excerpt:        excerpt,
		Content:        req.Content,
		CoverImagePath: req.CoverImagePath,
		IsPublished:    req.IsPublished,
	}
	if req.IsPublished {
		now := time.Now()
		post.PublishedAt = &now
	}

	if err := s.repo.Create(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *postService) UpdatePost(userID uint, id uint, req UpdatePostRequest) (*database.Post, error) {
	post, err := s.GetPostByID(userID, id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" && req.Title != post.Title {
		post.Title = req.Title
		post.Slug = s.uniqueSlug(req.Title, post.ID)
	}
	if strings.TrimSpace(req.Category) != "" {
		category, err := normalizeCategory(req.Category)
		if err != nil {
			return nil, err
		}
		post.Category = category
	}
	if req.Excerpt != "" {
		post.Excerpt = req.Excerpt
	}
	if req.Content != "" {
		post.Content = req.Content
	}
	if req.CoverImagePath != "" {
		if req.CoverImagePath == "DELETE" || req.CoverImagePath == "remove" {
			post.CoverImagePath = ""
		} else {
			post.CoverImagePath = req.CoverImagePath
		}
	}

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *postService) TogglePublish(userID uint, id uint) (*database.Post, error) {
	post, err := s.GetPostByID(userID, id)
	if err != nil {
		return nil, err
	}

	post.IsPublished = !post.IsPublished
	if post.IsPublished {
		now := time.Now()
		post.PublishedAt = &now
	} else {
		post.PublishedAt = nil
	}

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *postService) DeletePost(userID uint, id uint) error {
	if _, err := s.GetPostByID(userID, id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *postService) GetStats(userID uint) (*PostStatsResponse, error) {
	total, terbit, belumSelesai, err := s.repo.GetStatsByUserID(userID)
	if err != nil {
		return nil, err
	}
	return &PostStatsResponse{
		Total:        total,
		TotalTerbit:  terbit,
		BelumSelesai: belumSelesai,
	}, nil
}

func (s *postService) GetCategories() []string {
	return database.Categories
}

func (s *postService) GetPublicPosts(q, category string) ([]database.Post, error) {
	return s.repo.FindPublished(q, category)
}

func (s *postService) GetPublicPostBySlug(slug string) (*database.Post, error) {
	post, err := s.repo.FindPublishedBySlug(slug)
	if err != nil {
		return nil, ErrPostNotFound
	}
	return post, nil
}

func (s *postService) GetAllUsers() ([]UserWithStats, error) {
	return s.repo.FindUsersWithStats()
}

func (s *postService) GetAllPosts() ([]database.Post, error) {
	return s.repo.FindAll()
}

func (s *postService) GetUserPosts(targetUserID uint) ([]database.Post, error) {
	if _, err := s.repo.FindUserByID(targetUserID); err != nil {
		return nil, ErrUserNotFound
	}
	return s.repo.FindByUserID(targetUserID, "", "")
}
