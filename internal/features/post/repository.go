package post

import (
	"strings"

	"app/internal/database"
	"gorm.io/gorm"
)

type PostRepository interface {
	// Penulis
	FindByUserID(userID uint, q, category string) ([]database.Post, error)
	FindByID(id uint) (*database.Post, error)
	Create(post *database.Post) error
	Update(post *database.Post) error
	Delete(id uint) error
	SlugExists(slug string, excludeID uint) bool
	GetStatsByUserID(userID uint) (int64, int64, int64, error)

	// Publik
	FindPublished(q, category string) ([]database.Post, error)
	FindPublishedBySlug(slug string) (*database.Post, error)

	// Admin
	FindAll() ([]database.Post, error)
	FindUserByID(id uint) (*database.User, error)
	FindUsersWithStats() ([]UserWithStats, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

// applyFilters menambahkan pencarian kata kunci (judul/excerpt) dan filter kategori.
func applyFilters(query *gorm.DB, q, category string) *gorm.DB {
	q = strings.TrimSpace(q)
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ?", like, like)
	}
	category = strings.TrimSpace(category)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	return query
}

func (r *postRepository) FindByUserID(userID uint, q, category string) ([]database.Post, error) {
	var posts []database.Post
	query := applyFilters(r.db.Where("user_id = ?", userID), q, category)
	err := query.Order("created_at desc").Find(&posts).Error
	return posts, err
}

func (r *postRepository) FindByID(id uint) (*database.Post, error) {
	var post database.Post
	if err := r.db.First(&post, id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *postRepository) Create(post *database.Post) error {
	return r.db.Create(post).Error
}

func (r *postRepository) Update(post *database.Post) error {
	return r.db.Save(post).Error
}

func (r *postRepository) Delete(id uint) error {
	return r.db.Delete(&database.Post{}, id).Error
}

func (r *postRepository) SlugExists(slug string, excludeID uint) bool {
	var count int64
	r.db.Model(&database.Post{}).Where("slug = ? AND id <> ?", slug, excludeID).Count(&count)
	return count > 0
}

func (r *postRepository) GetStatsByUserID(userID uint) (int64, int64, int64, error) {
	var total, terbit, belumSelesai int64
	if err := r.db.Model(&database.Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, 0, 0, err
	}
	r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", userID, true).Count(&terbit)
	belumSelesai = total - terbit
	return total, terbit, belumSelesai, nil
}

func (r *postRepository) FindPublished(q, category string) ([]database.Post, error) {
	var posts []database.Post
	query := applyFilters(r.db.Where("is_published = ?", true), q, category)
	err := query.Preload("User").Order("published_at desc").Find(&posts).Error
	return posts, err
}

func (r *postRepository) FindPublishedBySlug(slug string) (*database.Post, error) {
	var post database.Post
	err := r.db.Where("slug = ? AND is_published = ?", slug, true).Preload("User").First(&post).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *postRepository) FindAll() ([]database.Post, error) {
	var posts []database.Post
	err := r.db.Preload("User").Order("created_at desc").Find(&posts).Error
	return posts, err
}

func (r *postRepository) FindUserByID(id uint) (*database.User, error) {
	var user database.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postRepository) FindUsersWithStats() ([]UserWithStats, error) {
	var users []database.User
	if err := r.db.Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}

	results := make([]UserWithStats, 0, len(users))
	for _, u := range users {
		var total, published int64
		r.db.Model(&database.Post{}).Where("user_id = ?", u.ID).Count(&total)
		r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", u.ID, true).Count(&published)
		results = append(results, UserWithStats{
			ID:             u.ID,
			Username:       u.Username,
			Email:          u.Email,
			Role:           u.Role,
			CreatedAt:      u.CreatedAt,
			TotalPosts:     total,
			TotalPublished: published,
		})
	}
	return results, nil
}
