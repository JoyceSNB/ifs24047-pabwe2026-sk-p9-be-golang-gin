package database

import (
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Categories adalah daftar kategori artikel blog yang valid.
var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// Post adalah model artikel blog yang dimigrasikan ke tabel posts.
type Post struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	Title          string     `gorm:"not null" json:"title"`
	Slug           string     `gorm:"uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"not null;default:'Umum';index" json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `gorm:"type:text" json:"content"`
	CoverImagePath string     `json:"cover_image_path,omitempty"`
	IsPublished    bool       `gorm:"default:false;index" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	ReadingTime    int        `gorm:"-" json:"reading_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	User           *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// AfterFind menghitung estimasi waktu baca setiap kali artikel dibaca dari database.
func (p *Post) AfterFind(_ *gorm.DB) error {
	p.ReadingTime = ReadingTime(p.Content)
	return nil
}

// AfterSave menghitung estimasi waktu baca setiap kali artikel dibuat atau diperbarui.
func (p *Post) AfterSave(_ *gorm.DB) error {
	p.ReadingTime = ReadingTime(p.Content)
	return nil
}

// ReadingTime menghitung estimasi waktu baca dalam menit (jumlah kata / 200 wpm, minimal 1).
func ReadingTime(content string) int {
	words := len(strings.Fields(content))
	minutes := (words + 199) / 200
	if minutes < 1 {
		return 1
	}
	return minutes
}

// Slugify mengubah judul menjadi slug URL (huruf kecil, angka, dan tanda hubung).
func Slugify(title string) string {
	slug := strings.Trim(nonAlphanumeric.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if slug == "" {
		return "artikel"
	}
	return slug
}

// NormalizeCategory mengembalikan nama kategori kanonik. ok bernilai false jika kategori tidak dikenal.
func NormalizeCategory(category string) (string, bool) {
	category = strings.TrimSpace(category)
	for _, c := range Categories {
		if strings.EqualFold(c, category) {
			return c, true
		}
	}
	return "", false
}
