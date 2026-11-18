package database

import (
	"fmt"
	"time"

	"app/internal/config"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// User model for database package migrations
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"not null;default:'member'" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBConnection == "sqlite" || cfg.DBConnection == "sqlite3" {
		dialector = sqlite.Open(cfg.DBDatabase)
	} else {
		dialector = postgres.Open(cfg.DSN())
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

func ConnectTest() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	_ = Migrate(db)
	return db, err
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Post{})
}

func Seed(db *gorm.DB, cfgs ...*config.Config) error {
	cfg := config.LoadConfig()
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	}
	adminUsername := cfg.AdminUsername
	adminPassword := cfg.AdminPassword

	// Seed Admin
	var admin User
	err := db.Where("username = ?", adminUsername).First(&admin).Error
	if err != nil {
		hashedAdmin, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		admin = User{
			Username: adminUsername,
			Email:    adminUsername + "@delcom.org",
			Password: string(hashedAdmin),
			Role:     "admin",
		}
		_ = db.Create(&admin).Error
	}

	// Seed Member
	var member User
	err = db.Where("username = ?", "member").First(&member).Error
	if err != nil {
		hashedMember, _ := bcrypt.GenerateFromPassword([]byte("member"), bcrypt.DefaultCost)
		member = User{
			Username: "member",
			Email:    "member@delcom.org",
			Password: string(hashedMember),
			Role:     "member",
		}
		_ = db.Create(&member).Error
	}

	// Seed artikel contoh jika tabel posts masih kosong
	var postCount int64
	db.Model(&Post{}).Count(&postCount)
	if postCount == 0 {
		seedPosts(db, member.ID)
	}

	return nil
}

type seedPost struct {
	Title     string
	Category  string
	Excerpt   string
	Content   string
	Published bool
}

func seedPosts(db *gorm.DB, authorID uint) {
	now := time.Now()
	posts := []seedPost{
		{"Mengenal Clean Architecture di Golang", "Teknologi", "Pengantar pemisahan handler, service, dan repository pada proyek Go.", "## Mengapa Clean Architecture?\n\nClean Architecture memisahkan tanggung jawab kode ke dalam lapisan yang jelas sehingga mudah diuji dan dirawat.\n\n## Lapisan Utama\n\n- Handler menerima request HTTP\n- Service menyimpan logika bisnis\n- Repository mengakses database", true},
		{"Tips Belajar React untuk Pemula", "Pendidikan", "Langkah-langkah praktis memulai belajar React dan Vite.", "## Mulai dari Dasar\n\nPahami komponen, props, dan state sebelum masuk ke hook yang lebih lanjut.\n\n## Latihan Rutin\n\nBuat proyek kecil seperti todolist atau blog sederhana untuk memperkuat pemahaman.", true},
		{"Membangun Kebiasaan Menulis Setiap Hari", "Gaya Hidup", "Cara menjaga konsistensi menulis agar menjadi kebiasaan.", "## Mulai dari Kecil\n\nTarget 100 kata per hari lebih realistis daripada satu artikel panjang seminggu sekali.\n\n## Catat Ide\n\nSimpan ide di catatan agar tidak hilang saat inspirasi datang.", true},
		{"Strategi Dasar Membangun Startup Digital", "Bisnis", "Gambaran awal validasi ide, produk minimum, dan umpan balik pengguna.", "## Validasi Ide\n\nBicaralah dengan calon pengguna sebelum menulis satu baris kode pun.\n\n## Produk Minimum\n\nRilis versi sederhana, ukur, lalu perbaiki berdasarkan data.", true},
		{"Apakah AI Akan Menggantikan Penulis?", "Opini", "Sudut pandang tentang peran AI dalam dunia penulisan konten.", "## Alat, Bukan Pengganti\n\nAI membantu menyusun draf, namun sudut pandang dan verifikasi fakta tetap peran penulis.\n\n## Kolaborasi\n\nPenulis yang memanfaatkan AI dengan bijak akan bekerja lebih efisien.", true},
		{"Catatan Praktikum PABWE Minggu Ini", "Umum", "Ringkasan pengerjaan praktikum pengembangan aplikasi berbasis web.", "## Ringkasan\n\nCatatan singkat tentang backend Golang Gin dan frontend React.\n\n## Rencana\n\nMenyelesaikan pengujian dan melengkapi dokumentasi.", false},
		{"Draf: Optimasi Query dengan Index", "Teknologi", "Draf artikel tentang indexing pada PostgreSQL.", "## Draf\n\nBagian ini masih dalam penulisan.", false},
	}
	for i, sp := range posts {
		post := Post{
			UserID:      authorID,
			Title:       sp.Title,
			Slug:        Slugify(sp.Title),
			Category:    sp.Category,
			Excerpt:     sp.Excerpt,
			Content:     sp.Content,
			IsPublished: sp.Published,
		}
		if sp.Published {
			publishedAt := now.Add(-time.Duration(i+1) * 24 * time.Hour)
			post.PublishedAt = &publishedAt
		}
		_ = db.Create(&post).Error
	}
}
