package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/api"
	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/middleware"
	"mainstory-digital-library-takehome/internal/service"
)

// BooksHandler serves catalog and library endpoints; role + optional auth drive visibility flags and content stripping.
type BooksHandler struct {
	svc *service.BookService
}

func NewBooksHandler(svc *service.BookService) *BooksHandler {
	return &BooksHandler{svc: svc}
}

// List is GET /books: paginated catalog rows with per-row access hints; works for guests and authenticated users.
func (h *BooksHandler) List(c *gin.Context) {
	// Nil UUID + empty role signals “guest” to BookService so optional auth can show LOCKED for every row.
	uid, uidOk := middleware.UserID(c)
	role, roleOk := middleware.Role(c)
	if !uidOk {
		uid = uuid.Nil
	}
	if !roleOk {
		role = ""
	}
	limit, offset, ok := parseLimitOffset(c)
	if !ok {
		return
	}
	filter, err := parseBookListFilter(c)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	items, err := h.svc.List(c.Request.Context(), uid, role, filter, limit, offset)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"books": items})
}

// RecentlyAdded is GET /books/recent — fixed top five by added_at; same JSON row shape and optional auth as List.
func (h *BooksHandler) RecentlyAdded(c *gin.Context) {
	uid, uidOk := middleware.UserID(c)
	role, roleOk := middleware.Role(c)
	if !uidOk {
		uid = uuid.Nil
	}
	if !roleOk {
		role = ""
	}
	items, err := h.svc.RecentlyAdded(c.Request.Context(), uid, role)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"books": items})
}

// MyLibrary is GET /users/me/library — subscription snapshot + purchased books without full text payloads.
func (h *BooksHandler) MyLibrary(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		api.WriteError(c, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}
	lib, err := h.svc.MyLibrary(c.Request.Context(), uid)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, lib)
}

// parseBookListFilter keeps query string parsing out of the service layer (HTTP-specific types and defaults).
func parseBookListFilter(c *gin.Context) (domain.BookListFilter, error) {
	var f domain.BookListFilter
	f.Q = strings.TrimSpace(c.Query("q"))
	f.Title = strings.TrimSpace(c.Query("title"))
	f.Author = strings.TrimSpace(c.Query("author"))
	f.Genre = strings.TrimSpace(c.Query("genre"))
	f.Language = strings.TrimSpace(c.Query("language"))
	if v := c.Query("is_fiction"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			// WHAT: reject non-boolean query values; reuses ErrInvalidBook so clients get a single validation bucket.
			return f, domain.ErrInvalidBook
		}
		f.IsFiction = &b
	}
	if v := c.Query("min_price_cents"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return f, domain.ErrInvalidCatalogFilters
		}
		x := int32(n)
		f.MinPriceCents = &x
	}
	if v := c.Query("max_price_cents"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return f, domain.ErrInvalidCatalogFilters
		}
		x := int32(n)
		f.MaxPriceCents = &x
	}
	return f, nil
}

// GetByID is GET /books/:id — returns metadata and may include content when caller is entitled or staff.
func (h *BooksHandler) GetByID(c *gin.Context) {
	uid, uidOk := middleware.UserID(c)
	role, roleOk := middleware.Role(c)
	if !uidOk {
		uid = uuid.Nil
	}
	if !roleOk {
		role = ""
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid book id")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), uid, role, id)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// createBookReq is the librarian/admin POST body; binding only enforces title (other fields default in handler).
type createBookReq struct {
	Title         string     `json:"title" binding:"required"`
	Description   string     `json:"description"`
	Author        string     `json:"author"`
	Genre         string     `json:"genre"`
	IsFiction     *bool      `json:"is_fiction"`
	PublishedDate *time.Time `json:"published_date"`
	Language      string     `json:"language"`
	PriceCents    int32      `json:"price_cents"`
	Content       string     `json:"content"`
}

func (h *BooksHandler) Create(c *gin.Context) {
	var req createBookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	// WHAT: default fiction=true and language=en when omitted so new rows stay valid for catalog display.
	isFiction := true
	if req.IsFiction != nil {
		isFiction = *req.IsFiction
	}
	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	b, err := h.svc.Create(c.Request.Context(), service.BookCreateInput{
		Title:         req.Title,
		Description:   req.Description,
		Author:        req.Author,
		Genre:         req.Genre,
		IsFiction:     isFiction,
		PublishedDate: req.PublishedDate,
		Language:      lang,
		PriceCents:    req.PriceCents,
		Content:       req.Content,
	})
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusCreated, b)
}

// updateBookReq matches create shape — PATCH replaces the full book payload per service/repository contract.
type updateBookReq struct {
	Title         string     `json:"title" binding:"required"`
	Description   string     `json:"description"`
	Author        string     `json:"author"`
	Genre         string     `json:"genre"`
	IsFiction     *bool      `json:"is_fiction"`
	PublishedDate *time.Time `json:"published_date"`
	Language      string     `json:"language"`
	PriceCents    int32      `json:"price_cents"`
	Content       string     `json:"content"`
}

// Update is PATCH /books/:id — same defaults as Create for fiction/language.
func (h *BooksHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid book id")
		return
	}
	var req updateBookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid JSON body")
		return
	}
	isFiction := true
	if req.IsFiction != nil {
		isFiction = *req.IsFiction
	}
	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	b, err := h.svc.Update(c.Request.Context(), id, service.BookUpdateInput{
		Title:         req.Title,
		Description:   req.Description,
		Author:        req.Author,
		Genre:         req.Genre,
		IsFiction:     isFiction,
		PublishedDate: req.PublishedDate,
		Language:      lang,
		PriceCents:    req.PriceCents,
		Content:       req.Content,
	})
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}

// Delete is admin-only route in main; 409 when purchases still reference the book.
func (h *BooksHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		api.WriteError(c, http.StatusBadRequest, "validation_error", "invalid book id")
		return
	}
	err = h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		api.WriteErrorFromDomain(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// parseLimitOffset reads list pagination from query; shared by books and entitlements handlers.
func parseLimitOffset(c *gin.Context) (limit, offset int32, ok bool) {
	limit = 50
	offset = 0
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			api.WriteError(c, http.StatusBadRequest, "validation_error", "limit must be between 1 and 100")
			return 0, 0, false
		}
		limit = int32(n)
	}
	if v := c.Query("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			api.WriteError(c, http.StatusBadRequest, "validation_error", "offset must be non-negative")
			return 0, 0, false
		}
		offset = int32(n)
	}
	return limit, offset, true
}
