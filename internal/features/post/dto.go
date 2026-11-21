package post

import (
	"time"

	"app/internal/database"
)

type CreatePostRequest struct {
	Title          string `json:"title" binding:"required"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

type UpdatePostRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
}

type PostStatsResponse struct {
	Total         int64 `json:"total"`
	TotalTerbit   int64 `json:"total_terbit"`
	BelumSelesai  int64 `json:"belum_selesai"`
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
}

type UserWithStats struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	TotalPosts     int64     `json:"total_posts"`
	TotalPublished int64     `json:"total_published"`
	TotalArticles  int64     `json:"total_articles"`
	TotalSelesai   int64     `json:"total_selesai"`
	BelumSelesai   int64     `json:"belum_selesai"`
}

// PublicAuthor hanya memuat data penulis yang aman ditampilkan ke publik.
type PublicAuthor struct {
	Username string `json:"username"`
}

// PublicPostResponse adalah bentuk artikel untuk endpoint publik: tanpa user_id,
// email, role, maupun data internal lain milik penulis.
type PublicPostResponse struct {
	ID             uint         `json:"id"`
	Title          string       `json:"title"`
	Slug           string       `json:"slug"`
	Category       string       `json:"category"`
	Excerpt        string       `json:"excerpt"`
	Content        string       `json:"content"`
	CoverImagePath string       `json:"cover_image_path,omitempty"`
	AuthorName     string       `json:"author_name"`
	User           PublicAuthor `json:"user"`
	ReadingTime    int          `json:"reading_time"`
	PublishedAt    *time.Time   `json:"published_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// ToPublicPost mengubah model Post menjadi respons publik yang aman.
func ToPublicPost(p database.Post) PublicPostResponse {
	author := ""
	if p.User != nil {
		author = p.User.Username
	}
	return PublicPostResponse{
		ID:             p.ID,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		AuthorName:     author,
		User:           PublicAuthor{Username: author},
		ReadingTime:    p.ReadingTime,
		PublishedAt:    p.PublishedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}