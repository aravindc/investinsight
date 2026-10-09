// Package category manages instrument categories and their sub-categories
// (e.g. Equity → Stock / ETF). Instruments reference a (category,
// sub-category) pair; Sector is a separate classification (internal/sector).
package category

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"investinsight/middleware/internal/sector"
)

// Other is the protected catch-all category, with a single protected "Other"
// sub-category; newly imported instruments default to it.
const Other = "Other"

type SubCategory struct {
	Name            string `json:"name"`
	InstrumentCount int    `json:"instrumentCount"`
}

type Category struct {
	Name            string        `json:"name"`
	InstrumentCount int           `json:"instrumentCount"`
	SubCategories   []SubCategory `json:"subCategories"`
}

var (
	ErrDuplicate = errors.New("already exists")
	ErrInUse     = errors.New("assigned to instruments")
	ErrNotFound  = errors.New("not found")
	// ErrNoCategory is returned when adding a sub-category to a missing category.
	ErrNoCategory = errors.New("category not found")
	ErrProtected  = errors.New(`"Other" can't be deleted`)
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List returns every category with its sub-categories, in display order.
func (r *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.name, s.name, COUNT(i.id)
		FROM categories c
		LEFT JOIN sub_categories s ON s.category = c.name
		LEFT JOIN instruments i ON i.category = s.category AND i.sub_category = s.name
		GROUP BY c.name, c.sort_order, s.name, s.sort_order
		ORDER BY c.sort_order, c.name, s.sort_order, s.name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var catName string
		var subName sql.NullString
		var count int
		if err := rows.Scan(&catName, &subName, &count); err != nil {
			return nil, err
		}
		if n := len(out); n == 0 || out[n-1].Name != catName {
			out = append(out, Category{Name: catName, SubCategories: []SubCategory{}})
		}
		cat := &out[len(out)-1]
		if subName.Valid {
			cat.SubCategories = append(cat.SubCategories, SubCategory{Name: subName.String, InstrumentCount: count})
			cat.InstrumentCount += count
		}
	}
	return out, rows.Err()
}

// CreateCategory adds a category after every existing one except "Other".
func (r *Repository) CreateCategory(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO categories (name, sort_order)
		SELECT $1, COALESCE(MAX(sort_order), 0) + 10 FROM categories WHERE name <> $2;`, name, Other)
	return mapErr(err)
}

// CreateSubCategory adds a sub-category at the end of category's list.
func (r *Repository) CreateSubCategory(ctx context.Context, category, name string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sub_categories (category, name, sort_order)
		SELECT $1::varchar, $2::varchar, COALESCE(MAX(sort_order), 0) + 10 FROM sub_categories WHERE category = $1::varchar;`, category, name)
	return mapErr(err)
}

// DeleteCategory removes a category and its sub-categories, provided no
// instrument uses any of them.
func (r *Repository) DeleteCategory(ctx context.Context, name string) error {
	if name == Other {
		return ErrProtected
	}
	return execDelete(r.db.ExecContext(ctx, `DELETE FROM categories WHERE name = $1;`, name))
}

// DeleteSubCategory removes a sub-category no instrument uses.
func (r *Repository) DeleteSubCategory(ctx context.Context, category, name string) error {
	if category == Other && name == Other {
		return ErrProtected
	}
	return execDelete(r.db.ExecContext(ctx,
		`DELETE FROM sub_categories WHERE category = $1 AND name = $2;`, category, name))
}

func execDelete(res sql.Result, err error) error {
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func mapErr(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505": // unique_violation (incl. case-insensitive indexes)
			return ErrDuplicate
		case "23503": // foreign_key_violation
			if pqErr.Constraint == "sub_categories_category_fkey" {
				return ErrNoCategory
			}
			return ErrInUse
		}
	}
	return err
}

type Handler struct {
	repo *Repository
	log  *zap.Logger
}

func NewHandler(repo *Repository, log *zap.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/categories", h.list)
	router.POST("/categories", h.createCategory)
	router.DELETE("/categories/:category", h.deleteCategory)
	router.POST("/categories/:category/sub-categories", h.createSubCategory)
	router.DELETE("/categories/:category/sub-categories/:subCategory", h.deleteSubCategory)
}

func (h *Handler) list(c *gin.Context) {
	cats, err := h.repo.List(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list categories", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list categories"})
		return
	}
	if cats == nil {
		cats = []Category{}
	}
	c.JSON(http.StatusOK, cats)
}

type nameRequest struct {
	Name string `json:"name"`
}

// bindName reads and normalizes {"name": …} with the same rules as sectors.
func bindName(c *gin.Context) (string, bool) {
	var req nameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return "", false
	}
	name, err := sector.NormalizeName(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return "", false
	}
	return name, true
}

func (h *Handler) createCategory(c *gin.Context) {
	name, ok := bindName(c)
	if !ok {
		return
	}
	h.respond(c, h.repo.CreateCategory(c.Request.Context(), name), http.StatusCreated, "category", gin.H{"name": name})
}

func (h *Handler) createSubCategory(c *gin.Context) {
	name, ok := bindName(c)
	if !ok {
		return
	}
	category := c.Param("category")
	h.respond(c, h.repo.CreateSubCategory(c.Request.Context(), category, name), http.StatusCreated, "sub-category",
		gin.H{"category": category, "name": name})
}

func (h *Handler) deleteCategory(c *gin.Context) {
	h.respond(c, h.repo.DeleteCategory(c.Request.Context(), c.Param("category")), http.StatusNoContent, "category", nil)
}

func (h *Handler) deleteSubCategory(c *gin.Context) {
	h.respond(c, h.repo.DeleteSubCategory(c.Request.Context(), c.Param("category"), c.Param("subCategory")),
		http.StatusNoContent, "sub-category", nil)
}

func (h *Handler) respond(c *gin.Context, err error, okStatus int, kind string, body any) {
	switch {
	case err == nil:
		if body == nil {
			c.Status(okStatus)
		} else {
			c.JSON(okStatus, body)
		}
	case errors.Is(err, ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "a " + kind + " with this name already exists"})
	case errors.Is(err, ErrInUse):
		c.JSON(http.StatusConflict, gin.H{"error": "Move its instruments elsewhere before deleting this " + kind + "."})
	case errors.Is(err, ErrProtected):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNoCategory):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": kind + " not found"})
	default:
		h.log.Error("category operation failed", zap.String("kind", kind), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update " + kind})
	}
}
