package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------- helpers ----------

type testEnv struct {
	handler  *PostHandler
	service  PostService
	repo     PostRepository
	db       *gorm.DB
	authorID uint
	otherID  uint
}

func setup(t *testing.T) *testEnv {
	db, err := database.ConnectTest()
	assert.NoError(t, err)

	author := database.User{Username: "author", Email: "author@example.com", Password: "x", Role: "member"}
	other := database.User{Username: "other", Email: "other@example.com", Password: "x", Role: "member"}
	assert.NoError(t, db.Create(&author).Error)
	assert.NoError(t, db.Create(&other).Error)

	repo := NewPostRepository(db)
	service := NewPostService(repo)
	return &testEnv{
		handler:  NewPostHandler(service),
		service:  service,
		repo:     repo,
		db:       db,
		authorID: author.ID,
		otherID:  other.ID,
	}
}

func doReq(h gin.HandlerFunc, method, path string, body any, userID uint, params ...gin.Param) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var buf *bytes.Buffer
	switch b := body.(type) {
	case nil:
		buf = bytes.NewBuffer(nil)
	case string:
		buf = bytes.NewBufferString(b)
	default:
		raw, _ := json.Marshal(b)
		buf = bytes.NewBuffer(raw)
	}

	c.Request, _ = http.NewRequest(method, path, buf)
	c.Request.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		c.Set("userID", userID)
	}
	c.Params = params
	h(c)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	var m map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	return m
}

func dataObj(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	obj, ok := decode(t, w)["data"].(map[string]any)
	assert.True(t, ok)
	return obj
}

func dataLen(t *testing.T, w *httptest.ResponseRecorder) int {
	if list, ok := decode(t, w)["data"].([]any); ok {
		return len(list)
	}
	return 0
}

func idParam(id uint) gin.Param {
	return gin.Param{Key: "id", Value: jsonNumber(id)}
}

func jsonNumber(id uint) string {
	b, _ := json.Marshal(id)
	return string(b)
}

// errRepo membungkus repository asli dan menyuntikkan error pada operasi tertentu.
type errRepo struct {
	PostRepository
	createErr error
	updateErr error
	statsErr  error
	listErr   error
}

func (e *errRepo) Create(p *database.Post) error {
	if e.createErr != nil {
		return e.createErr
	}
	return e.PostRepository.Create(p)
}

func (e *errRepo) Update(p *database.Post) error {
	if e.updateErr != nil {
		return e.updateErr
	}
	return e.PostRepository.Update(p)
}

func (e *errRepo) GetStatsByUserID(userID uint) (int64, int64, int64, error) {
	if e.statsErr != nil {
		return 0, 0, 0, e.statsErr
	}
	return e.PostRepository.GetStatsByUserID(userID)
}

func (e *errRepo) FindByUserID(userID uint, q, category string) ([]database.Post, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	return e.PostRepository.FindByUserID(userID, q, category)
}

func (e *errRepo) FindPublished(q, category string) ([]database.Post, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	return e.PostRepository.FindPublished(q, category)
}

func (e *errRepo) FindAll() ([]database.Post, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	return e.PostRepository.FindAll()
}

func (e *errRepo) FindUsersWithStats() ([]UserWithStats, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	return e.PostRepository.FindUsersWithStats()
}

// ---------- tests ----------

func TestPost_AuthorFlow(t *testing.T) {
	env := setup(t)
	h, uid, other := env.handler, env.authorID, env.otherID

	// 1. Create: artikel panjang (excerpt otomatis, reading time 3 menit), kategori dinormalisasi
	wA := doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{
		Title:    "Belajar Go Gin!",
		Category: "teknologi",
		Content:  strings.Repeat("kata ", 450),
	}, uid)
	assert.Equal(t, http.StatusCreated, wA.Code)
	dA := dataObj(t, wA)
	assert.Equal(t, "belajar-go-gin", dA["slug"])
	assert.Equal(t, "Teknologi", dA["category"])
	assert.Equal(t, float64(3), dA["reading_time"])
	assert.Equal(t, false, dA["is_published"])
	assert.True(t, strings.HasSuffix(dA["excerpt"].(string), "..."))
	idA := uint(dA["id"].(float64))

	// 2. Create: judul sama + langsung terbit -> slug unik dengan akhiran, excerpt singkat otomatis
	wB := doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{
		Title:       "Belajar Go Gin!",
		Category:    "Teknologi",
		Content:     "Isi singkat",
		IsPublished: true,
	}, uid)
	assert.Equal(t, http.StatusCreated, wB.Code)
	dB := dataObj(t, wB)
	assert.Equal(t, "belajar-go-gin-2", dB["slug"])
	assert.Equal(t, "Isi singkat", dB["excerpt"])
	assert.Equal(t, float64(1), dB["reading_time"])
	assert.Equal(t, true, dB["is_published"])
	assert.NotNil(t, dB["published_at"])
	idB := uint(dB["id"].(float64))

	// 3. Create: kategori kosong -> Umum, excerpt diberikan
	wC := doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{
		Title:   "Catatan Harian",
		Excerpt: "ringkasan khusus",
		Content: "Isi catatan",
	}, uid)
	assert.Equal(t, http.StatusCreated, wC.Code)
	dC := dataObj(t, wC)
	assert.Equal(t, "Umum", dC["category"])
	assert.Equal(t, "ringkasan khusus", dC["excerpt"])

	// 4. Create: error validasi
	wBadCat := doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{Title: "X", Category: "Olahraga", Content: "isi"}, uid)
	assert.Equal(t, http.StatusBadRequest, wBadCat.Code)
	wBadJSON := doReq(h.CreatePost, http.MethodPost, "/api/posts", "{invalid", uid)
	assert.Equal(t, http.StatusBadRequest, wBadJSON.Code)
	wNoTitle := doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{Content: "isi"}, uid)
	assert.Equal(t, http.StatusBadRequest, wNoTitle.Code)

	// 5. List + filter
	assert.Equal(t, 3, dataLen(t, doReq(h.GetPosts, http.MethodGet, "/api/posts", nil, uid)))
	assert.Equal(t, 1, dataLen(t, doReq(h.GetPosts, http.MethodGet, "/api/posts?q=catatan", nil, uid)))
	assert.Equal(t, 1, dataLen(t, doReq(h.GetPosts, http.MethodGet, "/api/posts?q=RINGKASAN", nil, uid)))
	assert.Equal(t, 2, dataLen(t, doReq(h.GetPosts, http.MethodGet, "/api/posts?category=Teknologi", nil, uid)))
	assert.Equal(t, 0, dataLen(t, doReq(h.GetPosts, http.MethodGet, "/api/posts?q=tidak-ada", nil, uid)))
	assert.Equal(t, 0, dataLen(t, doReq(h.GetPosts, http.MethodGet, "/api/posts", nil, other)))

	// 6. Detail
	wGet := doReq(h.GetPostByID, http.MethodGet, "/api/posts/1", nil, uid, idParam(idA))
	assert.Equal(t, http.StatusOK, wGet.Code)
	assert.Equal(t, http.StatusNotFound, doReq(h.GetPostByID, http.MethodGet, "/api/posts/9999", nil, uid, idParam(9999)).Code)
	assert.Equal(t, http.StatusForbidden, doReq(h.GetPostByID, http.MethodGet, "/api/posts/1", nil, other, idParam(idA)).Code)

	// 7. Update
	wUp := doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", UpdatePostRequest{
		Title:          "Judul Baru",
		Category:       "bisnis",
		Excerpt:        "Ringkasan baru",
		Content:        "isi baru",
		CoverImagePath: "cover.png",
	}, uid, idParam(idA))
	assert.Equal(t, http.StatusOK, wUp.Code)
	dUp := dataObj(t, wUp)
	assert.Equal(t, "judul-baru", dUp["slug"])
	assert.Equal(t, "Bisnis", dUp["category"])
	assert.Equal(t, "Ringkasan baru", dUp["excerpt"])
	assert.Equal(t, "isi baru", dUp["content"])
	assert.Equal(t, "cover.png", dUp["cover_image_path"])
	assert.Equal(t, float64(1), dUp["reading_time"])

	// Judul sama persis -> slug tidak berubah
	wSame := doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", UpdatePostRequest{Title: "Judul Baru"}, uid, idParam(idA))
	assert.Equal(t, "judul-baru", dataObj(t, wSame)["slug"])

	// Judul berbeda tetapi slug sama dengan miliknya sendiri -> tetap "judul-baru"
	wSelf := doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", UpdatePostRequest{Title: "Judul-Baru!"}, uid, idParam(idA))
	assert.Equal(t, "judul-baru", dataObj(t, wSelf)["slug"])

	// Hapus cover image
	wDel := doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", UpdatePostRequest{CoverImagePath: "DELETE"}, uid, idParam(idA))
	assert.Equal(t, http.StatusOK, wDel.Code)
	assert.Nil(t, dataObj(t, wDel)["cover_image_path"])

	// Update kosong -> tidak ada perubahan
	wEmpty := doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", UpdatePostRequest{}, uid, idParam(idA))
	assert.Equal(t, http.StatusOK, wEmpty.Code)
	assert.Equal(t, "Judul-Baru!", dataObj(t, wEmpty)["title"])

	// Update gagal
	assert.Equal(t, http.StatusBadRequest, doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", UpdatePostRequest{Category: "Salah"}, uid, idParam(idA)).Code)
	assert.Equal(t, http.StatusBadRequest, doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", "{invalid", uid, idParam(idA)).Code)
	assert.Equal(t, http.StatusNotFound, doReq(h.UpdatePost, http.MethodPut, "/api/posts/9999", UpdatePostRequest{}, uid, idParam(9999)).Code)
	assert.Equal(t, http.StatusForbidden, doReq(h.UpdatePost, http.MethodPut, "/api/posts/1", UpdatePostRequest{}, other, idParam(idA)).Code)

	// 8. Toggle publish: draf -> terbit -> draf
	wT1 := doReq(h.TogglePublish, http.MethodPatch, "/api/posts/1/toggle-publish", nil, uid, idParam(idA))
	assert.Equal(t, http.StatusOK, wT1.Code)
	dT1 := dataObj(t, wT1)
	assert.Equal(t, true, dT1["is_published"])
	assert.NotNil(t, dT1["published_at"])

	wT2 := doReq(h.TogglePublish, http.MethodPatch, "/api/posts/1/toggle-publish", nil, uid, idParam(idA))
	dT2 := dataObj(t, wT2)
	assert.Equal(t, false, dT2["is_published"])
	assert.Nil(t, dT2["published_at"])

	assert.Equal(t, http.StatusNotFound, doReq(h.TogglePublish, http.MethodPatch, "/x", nil, uid, idParam(9999)).Code)
	assert.Equal(t, http.StatusForbidden, doReq(h.TogglePublish, http.MethodPatch, "/x", nil, other, idParam(idA)).Code)

	// 9. Stats: 3 artikel, 1 terbit (B), 2 draf
	wStats := doReq(h.GetStats, http.MethodGet, "/api/posts/stats", nil, uid)
	assert.Equal(t, http.StatusOK, wStats.Code)
	stats := dataObj(t, wStats)
	assert.Equal(t, float64(3), stats["total"])
	assert.Equal(t, float64(1), stats["total_terbit"])
	assert.Equal(t, float64(2), stats["belum_selesai"])

	// 10. Delete
	assert.Equal(t, http.StatusForbidden, doReq(h.DeletePost, http.MethodDelete, "/x", nil, other, idParam(idB)).Code)
	assert.Equal(t, http.StatusOK, doReq(h.DeletePost, http.MethodDelete, "/x", nil, uid, idParam(idB)).Code)
	assert.Equal(t, http.StatusNotFound, doReq(h.DeletePost, http.MethodDelete, "/x", nil, uid, idParam(idB)).Code)
}

func TestPost_HandlerGuards(t *testing.T) {
	env := setup(t)
	h := env.handler

	// Tanpa userID -> 401
	handlers := map[string]gin.HandlerFunc{
		"GetPosts":      h.GetPosts,
		"GetPostByID":   h.GetPostByID,
		"CreatePost":    h.CreatePost,
		"UpdatePost":    h.UpdatePost,
		"TogglePublish": h.TogglePublish,
		"DeletePost":    h.DeletePost,
		"GetStats":      h.GetStats,
	}
	for name, fn := range handlers {
		w := doReq(fn, http.MethodGet, "/x", nil, 0, idParam(1))
		assert.Equal(t, http.StatusUnauthorized, w.Code, name)
	}

	// ID tidak valid -> 400
	bad := gin.Param{Key: "id", Value: "abc"}
	idHandlers := map[string]gin.HandlerFunc{
		"GetPostByID":   h.GetPostByID,
		"UpdatePost":    h.UpdatePost,
		"TogglePublish": h.TogglePublish,
		"DeletePost":    h.DeletePost,
		"GetUserPosts":  h.GetUserPosts,
	}
	for name, fn := range idHandlers {
		w := doReq(fn, http.MethodGet, "/x", nil, env.authorID, bad)
		assert.Equal(t, http.StatusBadRequest, w.Code, name)
	}
}

func TestPost_PublicEndpoints(t *testing.T) {
	env := setup(t)
	h, uid := env.handler, env.authorID

	// Satu draf dan satu terbit
	doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{Title: "Artikel Draf", Category: "Umum", Content: "isi draf"}, uid)
	doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{Title: "Artikel Terbit Gin", Category: "Teknologi", Content: "isi terbit", IsPublished: true}, uid)

	// Kategori
	wCat := doReq(h.GetCategories, http.MethodGet, "/api/public/categories", nil, 0)
	assert.Equal(t, http.StatusOK, wCat.Code)
	assert.Equal(t, 6, dataLen(t, wCat))

	// Daftar publik hanya artikel terbit
	assert.Equal(t, 1, dataLen(t, doReq(h.GetPublicPosts, http.MethodGet, "/api/public/posts", nil, 0)))
	assert.Equal(t, 1, dataLen(t, doReq(h.GetPublicPosts, http.MethodGet, "/api/public/posts?q=gin", nil, 0)))
	assert.Equal(t, 1, dataLen(t, doReq(h.GetPublicPosts, http.MethodGet, "/api/public/posts?category=Teknologi", nil, 0)))
	assert.Equal(t, 0, dataLen(t, doReq(h.GetPublicPosts, http.MethodGet, "/api/public/posts?category=Umum", nil, 0)))

	// Detail publik berdasarkan slug (lengkap dengan penulis)
	wSlug := doReq(h.GetPublicPostBySlug, http.MethodGet, "/api/public/posts/x", nil, 0, gin.Param{Key: "slug", Value: "artikel-terbit-gin"})
	assert.Equal(t, http.StatusOK, wSlug.Code)
	d := dataObj(t, wSlug)
	assert.Equal(t, "Artikel Terbit Gin", d["title"])
	assert.Equal(t, "author", d["user"].(map[string]any)["username"])

	// Draf dan slug tidak dikenal -> 404
	wDraft := doReq(h.GetPublicPostBySlug, http.MethodGet, "/x", nil, 0, gin.Param{Key: "slug", Value: "artikel-draf"})
	assert.Equal(t, http.StatusNotFound, wDraft.Code)
	wNone := doReq(h.GetPublicPostBySlug, http.MethodGet, "/x", nil, 0, gin.Param{Key: "slug", Value: "tidak-ada"})
	assert.Equal(t, http.StatusNotFound, wNone.Code)
}

func TestPost_AdminEndpoints(t *testing.T) {
	env := setup(t)
	h, uid := env.handler, env.authorID

	doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{Title: "Satu", Content: "isi", IsPublished: true}, uid)
	doReq(h.CreatePost, http.MethodPost, "/api/posts", CreatePostRequest{Title: "Dua", Content: "isi"}, uid)

	// Pengguna beserta total artikel
	wUsers := doReq(h.GetAllUsers, http.MethodGet, "/api/admin/users", nil, 0)
	assert.Equal(t, http.StatusOK, wUsers.Code)
	var found bool
	for _, item := range decode(t, wUsers)["data"].([]any) {
		u := item.(map[string]any)
		if u["username"] == "author" {
			found = true
			assert.Equal(t, float64(2), u["total_posts"])
			assert.Equal(t, float64(1), u["total_published"])
		}
	}
	assert.True(t, found)

	// Arsip seluruh artikel
	assert.Equal(t, 2, dataLen(t, doReq(h.GetAllPosts, http.MethodGet, "/api/admin/posts", nil, 0)))

	// Artikel milik pengguna tertentu
	wUser := doReq(h.GetUserPosts, http.MethodGet, "/x", nil, 0, idParam(uid))
	assert.Equal(t, http.StatusOK, wUser.Code)
	assert.Equal(t, 2, dataLen(t, wUser))
	assert.Equal(t, http.StatusNotFound, doReq(h.GetUserPosts, http.MethodGet, "/x", nil, 0, idParam(9999)).Code)
}

func TestPost_ServiceAndHandlerErrors(t *testing.T) {
	env := setup(t)
	uid := env.authorID

	// Artikel awal lewat repository asli
	base := &database.Post{UserID: uid, Title: "Awal", Slug: "awal", Category: "Umum", Content: "isi"}
	assert.NoError(t, env.repo.Create(base))

	boom := errors.New("db down")
	svc := NewPostService(&errRepo{
		PostRepository: env.repo,
		createErr:      boom,
		updateErr:      boom,
		statsErr:       boom,
		listErr:        boom,
	})
	h := NewPostHandler(svc)

	// Service
	_, err := svc.CreatePost(uid, CreatePostRequest{Title: "Baru", Content: "isi"})
	assert.ErrorIs(t, err, boom)
	_, err = svc.UpdatePost(uid, base.ID, UpdatePostRequest{Content: "baru"})
	assert.ErrorIs(t, err, boom)
	_, err = svc.TogglePublish(uid, base.ID)
	assert.ErrorIs(t, err, boom)
	_, err = svc.GetStats(uid)
	assert.ErrorIs(t, err, boom)

	// Handler -> 500
	assert.Equal(t, http.StatusInternalServerError, doReq(h.CreatePost, http.MethodPost, "/x", CreatePostRequest{Title: "Baru", Content: "isi"}, uid).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(h.UpdatePost, http.MethodPut, "/x", UpdatePostRequest{Content: "baru"}, uid, idParam(base.ID)).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(h.TogglePublish, http.MethodPatch, "/x", nil, uid, idParam(base.ID)).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(h.GetStats, http.MethodGet, "/x", nil, uid).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(h.GetPosts, http.MethodGet, "/x", nil, uid).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(h.GetPublicPosts, http.MethodGet, "/x", nil, 0).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(h.GetAllPosts, http.MethodGet, "/x", nil, 0).Code)
	assert.Equal(t, http.StatusInternalServerError, doReq(h.GetAllUsers, http.MethodGet, "/x", nil, 0).Code)
}

func TestPost_RepositoryErrors(t *testing.T) {
	env := setup(t)

	// Tabel posts hilang -> statistik error
	assert.NoError(t, env.db.Migrator().DropTable(&database.Post{}))
	_, _, _, err := env.repo.GetStatsByUserID(env.authorID)
	assert.Error(t, err)

	// Tabel users hilang -> daftar pengguna error
	assert.NoError(t, env.db.Migrator().DropTable(&database.User{}))
	_, err = env.repo.FindUsersWithStats()
	assert.Error(t, err)
}
