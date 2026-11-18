package database

import (
	"strings"
	"testing"

	"app/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConnectTest_And_Migrate_And_Seed(t *testing.T) {
	db, err := ConnectTest()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Test Seeding with config
	cfg := &config.Config{
		AdminUsername: "admin",
		AdminPassword: "admin",
	}
	err = Seed(db, cfg)
	assert.NoError(t, err)

	// Verify Admin
	var admin User
	err = db.Where("username = ?", "admin").First(&admin).Error
	assert.NoError(t, err)
	assert.Equal(t, "admin", admin.Role)

	// Verify Member
	var member User
	err = db.Where("username = ?", "member").First(&member).Error
	assert.NoError(t, err)
	assert.Equal(t, "member", member.Role)

	// Calling Seed again without config should be idempotent
	err = Seed(db)
	assert.NoError(t, err)

	var userCount int64
	db.Model(&User{}).Count(&userCount)
	assert.Equal(t, int64(2), userCount)

	// Verify seeded posts (5 terbit, 2 draf)
	var totalPosts int64
	db.Model(&Post{}).Count(&totalPosts)
	assert.Equal(t, int64(7), totalPosts)

	var publishedPosts int64
	db.Model(&Post{}).Where("is_published = ?", true).Count(&publishedPosts)
	assert.Equal(t, int64(5), publishedPosts)

	var published Post
	assert.NoError(t, db.Where("slug = ?", "mengenal-clean-architecture-di-golang").First(&published).Error)
	assert.NotNil(t, published.PublishedAt)
	assert.GreaterOrEqual(t, published.ReadingTime, 1)

	var draft Post
	assert.NoError(t, db.Where("is_published = ?", false).First(&draft).Error)
	assert.Nil(t, draft.PublishedAt)
}

func TestPostHelpers(t *testing.T) {
	// ReadingTime
	assert.Equal(t, 1, ReadingTime(""))
	assert.Equal(t, 1, ReadingTime("hanya beberapa kata"))
	assert.Equal(t, 1, ReadingTime(strings.Repeat("kata ", 200)))
	assert.Equal(t, 2, ReadingTime(strings.Repeat("kata ", 201)))
	assert.Equal(t, 3, ReadingTime(strings.Repeat("kata ", 450)))

	// Slugify
	assert.Equal(t, "belajar-go-gin", Slugify("Belajar Go Gin!"))
	assert.Equal(t, "tips-trik-2026", Slugify("  Tips & Trik 2026  "))
	assert.Equal(t, "artikel", Slugify("!!!"))
	assert.Equal(t, "artikel", Slugify(""))

	// NormalizeCategory
	c, ok := NormalizeCategory("teknologi")
	assert.True(t, ok)
	assert.Equal(t, "Teknologi", c)
	c, ok = NormalizeCategory("  Gaya Hidup ")
	assert.True(t, ok)
	assert.Equal(t, "Gaya Hidup", c)
	_, ok = NormalizeCategory("Olahraga")
	assert.False(t, ok)
	_, ok = NormalizeCategory("")
	assert.False(t, ok)
}

func TestPost_Hooks(t *testing.T) {
	db, err := ConnectTest()
	assert.NoError(t, err)

	user := User{Username: "hook_user", Email: "hook@example.com", Password: "x", Role: "member"}
	assert.NoError(t, db.Create(&user).Error)

	post := Post{
		UserID:   user.ID,
		Title:    "Artikel Panjang",
		Slug:     "artikel-panjang",
		Category: "Umum",
		Content:  strings.Repeat("kata ", 450),
	}
	assert.NoError(t, db.Create(&post).Error)
	assert.Equal(t, 3, post.ReadingTime) // AfterSave

	var found Post
	assert.NoError(t, db.First(&found, post.ID).Error)
	assert.Equal(t, 3, found.ReadingTime) // AfterFind

	found.Content = "singkat"
	assert.NoError(t, db.Save(&found).Error)
	assert.Equal(t, 1, found.ReadingTime)
}

func TestConnect_Sqlite(t *testing.T) {
	cfg := &config.Config{
		DBConnection: "sqlite",
		DBDatabase:   ":memory:",
	}
	db, err := Connect(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	err = Migrate(db)
	assert.NoError(t, err)
}

func TestConnect_Postgres_Invalid(t *testing.T) {
	// Connecting to non-existent postgres port should return error or fail gracefully
	cfg := &config.Config{
		DBConnection: "pgsql",
		DBHost:       "127.0.0.1",
		DBPort:       "54399",
		DBDatabase:   "invalid_db",
		DBUsername:   "invalid_user",
		DBPassword:   "invalid_pass",
		DBSSLMode:    "disable",
	}
	db, err := Connect(cfg)
	if err == nil && db != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}
