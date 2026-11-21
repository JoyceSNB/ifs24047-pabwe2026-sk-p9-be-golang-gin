package post

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/response"
	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	service PostService
}

func NewPostHandler(service PostService) *PostHandler {
	return &PostHandler{service: service}
}

func getUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("userID")
	if !exists {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return 0, false
	}
	return val.(uint), true
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid ID parameter")
		return 0, false
	}
	return uint(id), true
}

// respondError memetakan error service ke status HTTP yang sesuai.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPostNotFound), errors.Is(err, ErrUserNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Fail(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidCategory):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, err.Error())
	}
}

// ===== Publik =====

func (h *PostHandler) GetCategories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Categories retrieved successfully", h.service.GetCategories())
}

func (h *PostHandler) GetPublicPosts(c *gin.Context) {
	posts, err := h.service.GetPublicPosts(c.Query("q"), c.Query("category"))
	if err != nil {
		respondError(c, err)
		return
	}
	items := make([]PublicPostResponse, 0, len(posts))
	for _, p := range posts {
		items = append(items, ToPublicPost(p))
	}
	response.Success(c, http.StatusOK, "Posts retrieved successfully", items)
}

func (h *PostHandler) GetPublicPostBySlug(c *gin.Context) {
	post, err := h.service.GetPublicPostBySlug(c.Param("slug"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved successfully", ToPublicPost(*post))
}

// ===== Penulis =====

func (h *PostHandler) GetPosts(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	posts, err := h.service.GetPosts(userID, c.Query("q"), c.Query("category"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Posts retrieved successfully", posts)
}

func (h *PostHandler) GetPostByID(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	post, err := h.service.GetPostByID(userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved successfully", post)
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	post, err := h.service.CreatePost(userID, req)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Post created successfully", post)
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	post, err := h.service.UpdatePost(userID, id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post updated successfully", post)
}

func (h *PostHandler) TogglePublish(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	post, err := h.service.TogglePublish(userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post publish status toggled successfully", post)
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeletePost(userID, id); err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post deleted successfully")
}

func (h *PostHandler) GetStats(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	stats, err := h.service.GetStats(userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Stats retrieved successfully", stats)
}

// ===== Admin =====

func (h *PostHandler) GetAllUsers(c *gin.Context) {
	users, err := h.service.GetAllUsers()
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Users retrieved successfully", users)
}

func (h *PostHandler) GetAllPosts(c *gin.Context) {
	posts, err := h.service.GetAllPosts()
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Posts retrieved successfully", posts)
}

func (h *PostHandler) GetUserPosts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	posts, err := h.service.GetUserPosts(id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User posts retrieved successfully", posts)
}