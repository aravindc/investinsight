// Package sector manages the sectors instruments are classified into. They
// live in the sectors table (seeded with the original fixed list) so new
// ones can be created from the Admin page; instruments.sector references a
// sector by name.
package sector

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// Other is the catch-all sector, always listed last and never deletable.
const Other = "Other"

const maxNameLength = 50

// SubSector is a subdivision of a sector and how many instruments use it.
type SubSector struct {
	Name            string `json:"name"`
	InstrumentCount int    `json:"instrumentCount"`
}

// Sector is a sector, how many instruments are assigned to it, and its
// sub-sectors.
type Sector struct {
	Name            string      `json:"name"`
	InstrumentCount int         `json:"instrumentCount"`
	SubSectors      []SubSector `json:"subSectors"`
}

var (
	ErrDuplicate = errors.New("a sector with this name already exists")
	ErrInUse     = errors.New("sector is assigned to instruments")
	ErrNotFound  = errors.New("sector not found")
	ErrProtected = errors.New(`the "Other" sector can't be deleted`)
	// ErrNoSector is returned when adding a sub-sector to a missing sector.
	ErrNoSector = errors.New("sector not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List returns every sector with its sub-sectors, in display order.
func (r *Repository) List(ctx context.Context) ([]Sector, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.name,
			(SELECT COUNT(*) FROM instruments i WHERE i.sector = s.name),
			ss.name,
			(SELECT COUNT(*) FROM instruments i WHERE i.sector = ss.sector AND i.sub_sector = ss.name)
		FROM sectors s
		LEFT JOIN sub_sectors ss ON ss.sector = s.name
		ORDER BY s.sort_order, s.name, ss.name;`) // sub-sectors alphabetically
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Sector
	for rows.Next() {
		var name string
		var count, subCount int
		var subName sql.NullString
		if err := rows.Scan(&name, &count, &subName, &subCount); err != nil {
			return nil, err
		}
		if n := len(out); n == 0 || out[n-1].Name != name {
			out = append(out, Sector{Name: name, InstrumentCount: count, SubSectors: []SubSector{}})
		}
		if subName.Valid {
			last := &out[len(out)-1]
			last.SubSectors = append(last.SubSectors, SubSector{Name: subName.String, InstrumentCount: subCount})
		}
	}
	return out, rows.Err()
}

// CreateSubSector adds a sub-sector at the end of sector's list.
func (r *Repository) CreateSubSector(ctx context.Context, sectorName, name string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sub_sectors (sector, name, sort_order)
		SELECT $1::varchar, $2::varchar, COALESCE(MAX(sort_order), 0) + 10 FROM sub_sectors WHERE sector = $1::varchar;`,
		sectorName, name)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return ErrDuplicate
		case "23503":
			return ErrNoSector
		}
	}
	return err
}

// DeleteSubSector removes a sub-sector no instrument uses.
func (r *Repository) DeleteSubSector(ctx context.Context, sectorName, name string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sub_sectors WHERE sector = $1 AND name = $2;`, sectorName, name)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23503" {
		return ErrInUse
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Create adds a sector after every existing one except "Other".
func (r *Repository) Create(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sectors (name, sort_order)
		SELECT $1, COALESCE(MAX(sort_order), 0) + 10 FROM sectors WHERE name <> $2;`, name, Other)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" { // unique_violation (incl. case-insensitive index)
		return ErrDuplicate
	}
	return err
}

// Delete removes a sector no instrument uses.
func (r *Repository) Delete(ctx context.Context, name string) error {
	if name == Other {
		return ErrProtected
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM sectors WHERE name = $1;`, name)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23503" { // foreign_key_violation
		return ErrInUse
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// NormalizeName trims and collapses internal whitespace, and checks length.
func NormalizeName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	switch {
	case name == "":
		return "", errors.New("sector name is required")
	case utf8.RuneCountInString(name) > maxNameLength:
		return "", errors.New("sector name must be 50 characters or fewer")
	case strings.Contains(name, "/"):
		// The name is a URL path segment in DELETE /sectors/:name.
		return "", errors.New(`sector name can't contain "/"`)
	}
	return name, nil
}

type Handler struct {
	repo *Repository
	log  *zap.Logger
}

func NewHandler(repo *Repository, log *zap.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/sectors", h.list)
	router.POST("/sectors", h.create)
	router.DELETE("/sectors/:name", h.delete)
	router.POST("/sectors/:name/sub-sectors", h.createSubSector)
	router.DELETE("/sectors/:name/sub-sectors/:subSector", h.deleteSubSector)
}

func (h *Handler) createSubSector(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	name, err := NormalizeName(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": strings.Replace(err.Error(), "sector name", "sub-sector name", 1)})
		return
	}
	sectorName := c.Param("name")
	switch err := h.repo.CreateSubSector(c.Request.Context(), sectorName, name); {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"sector": sectorName, "name": name})
	case errors.Is(err, ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "a sub-sector with this name already exists in this sector"})
	case errors.Is(err, ErrNoSector):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		h.log.Error("failed to create sub-sector", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sub-sector"})
	}
}

func (h *Handler) deleteSubSector(c *gin.Context) {
	switch err := h.repo.DeleteSubSector(c.Request.Context(), c.Param("name"), c.Param("subSector")); {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInUse):
		c.JSON(http.StatusConflict, gin.H{"error": "Move its instruments to another sub-sector before deleting it."})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "sub-sector not found"})
	default:
		h.log.Error("failed to delete sub-sector", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete sub-sector"})
	}
}

func (h *Handler) list(c *gin.Context) {
	sectors, err := h.repo.List(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list sectors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sectors"})
		return
	}
	if sectors == nil {
		sectors = []Sector{}
	}
	c.JSON(http.StatusOK, sectors)
}

type createRequest struct {
	Name string `json:"name"`
}

func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	name, err := NormalizeName(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch err := h.repo.Create(c.Request.Context(), name); {
	case err == nil:
		h.log.Info("sector created", zap.String("name", name))
		c.JSON(http.StatusCreated, Sector{Name: name})
	case errors.Is(err, ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		h.log.Error("failed to create sector", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sector"})
	}
}

func (h *Handler) delete(c *gin.Context) {
	name := c.Param("name")
	switch err := h.repo.Delete(c.Request.Context(), name); {
	case err == nil:
		h.log.Info("sector deleted", zap.String("name", name))
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInUse):
		c.JSON(http.StatusConflict, gin.H{"error": "Move its instruments to another sector before deleting it."})
	case errors.Is(err, ErrProtected):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		h.log.Error("failed to delete sector", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete sector"})
	}
}
