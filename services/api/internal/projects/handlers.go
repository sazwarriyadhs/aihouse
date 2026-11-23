package projects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

type createRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	LandWidth   float64 `json:"land_width_m"`
	LandDepth   float64 `json:"land_depth_m"`
	Orientation string  `json:"orientation"`
	Floors      int     `json:"floors"`
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/projects" {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			respond(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/v1/projects/") {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			respond(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		h.get(w, r, strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"))
		return
	}

	respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req createRequest
	if err := decoder.Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respond(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON object"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Location = strings.TrimSpace(req.Location)
	req.Orientation = strings.ToUpper(strings.TrimSpace(req.Orientation))

	if req.Name == "" || len(req.Name) > 200 {
		respond(w, http.StatusBadRequest, map[string]string{"error": "name is required and must not exceed 200 characters"})
		return
	}
	if len(req.Description) > 5000 || len(req.Location) > 300 {
		respond(w, http.StatusBadRequest, map[string]string{"error": "description or location is too long"})
		return
	}
	if math.IsNaN(req.LandWidth) || math.IsInf(req.LandWidth, 0) ||
		math.IsNaN(req.LandDepth) || math.IsInf(req.LandDepth, 0) ||
		req.LandWidth < 1 || req.LandWidth > 1000 ||
		req.LandDepth < 1 || req.LandDepth > 1000 {
		respond(w, http.StatusBadRequest, map[string]string{"error": "land dimensions must be between 1 and 1000 metres"})
		return
	}
	if req.Floors < 1 || req.Floors > 2 {
		respond(w, http.StatusBadRequest, map[string]string{"error": "floors must be 1 or 2"})
		return
	}
	switch req.Orientation {
	case "N", "E", "S", "W", "NE", "NW", "SE", "SW":
	default:
		respond(w, http.StatusBadRequest, map[string]string{"error": "orientation must be N, NE, E, SE, S, SW, W or NW"})
		return
	}

	area := math.Round(req.LandWidth*req.LandDepth*100) / 100
	projectID := uuid.New()
	designID := uuid.New()

	model := map[string]any{
		"schema_version": "1.0",
		"project": map[string]any{
			"name":        req.Name,
			"description": req.Description,
		},
		"site": map[string]any{
			"location": req.Location,
			"land": map[string]any{
				"shape":       "rectangle",
				"width_m":     req.LandWidth,
				"depth_m":     req.LandDepth,
				"area_m2":     area,
				"orientation": req.Orientation,
			},
		},
		"building": map[string]any{
			"floors": req.Floors,
		},
		"units": map[string]any{
			"length": "m",
			"area":   "m2",
		},
	}

	modelJSON, err := json.Marshal(model)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not encode design model"})
		return
	}
	sum := sha256.Sum256(modelJSON)
	checksum := hex.EncodeToString(sum[:])

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not start database transaction"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	_, err = tx.Exec(r.Context(),
		`INSERT INTO core.projects (id, name, description) VALUES ($1, $2, $3)`,
		projectID, req.Name, req.Description)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not create project"})
		return
	}

	_, err = tx.Exec(r.Context(),
		`INSERT INTO core.designs (id, project_id, name, current_version, status)
         VALUES ($1, $2, $3, 1, 'draft')`,
		designID, projectID, req.Name+" - Initial Design")
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not create initial design"})
		return
	}

	var versionID uuid.UUID
	err = tx.QueryRow(r.Context(),
		`INSERT INTO core.design_versions
         (design_id, version_number, canonical_model, model_checksum)
         VALUES ($1, 1, $2::jsonb, $3)
         RETURNING id`,
		designID, string(modelJSON), checksum).Scan(&versionID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not save initial design version"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not commit project"})
		return
	}

	respond(w, http.StatusCreated, map[string]any{
		"project_id":      projectID,
		"design_id":       designID,
		"version_id":      versionID,
		"canonical_model": json.RawMessage(modelJSON),
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, description, status, created_at, updated_at
         FROM core.projects
         ORDER BY created_at DESC`)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not list projects"})
		return
	}
	defer rows.Close()

	type project struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		CreatedAt   any    `json:"created_at"`
		UpdatedAt   any    `json:"updated_at"`
	}
	items := make([]project, 0)

	for rows.Next() {
		var item project
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read projects"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read projects"})
		return
	}
	respond(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid project ID"})
		return
	}

	var item struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		CreatedAt   any    `json:"created_at"`
		UpdatedAt   any    `json:"updated_at"`
	}
	err = h.pool.QueryRow(r.Context(),
		`SELECT id, name, description, status, created_at, updated_at
         FROM core.projects WHERE id = $1`, id).
		Scan(&item.ID, &item.Name, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		respond(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read project"})
		return
	}

	var designs []map[string]any
	rows, err := h.pool.Query(r.Context(),
		`SELECT d.id, d.name, d.current_version, d.status,
                v.id, v.version_number, v.canonical_model, v.model_checksum
         FROM core.designs d
         LEFT JOIN core.design_versions v
           ON v.design_id = d.id AND v.version_number = d.current_version
         WHERE d.project_id = $1
         ORDER BY d.created_at`, id)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read project designs"})
		return
	}
	defer rows.Close()

	designs = make([]map[string]any, 0)
	for rows.Next() {
		var designID uuid.UUID
		var designName, designStatus string
		var currentVersion int
		var versionID *uuid.UUID
		var versionNumber *int
		var model []byte
		var checksum *string

		if err := rows.Scan(&designID, &designName, &currentVersion, &designStatus,
			&versionID, &versionNumber, &model, &checksum); err != nil {
			respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read design version"})
			return
		}

		design := map[string]any{
			"id":              designID,
			"name":            designName,
			"current_version": currentVersion,
			"status":          designStatus,
		}
		if versionID != nil {
			design["version_id"] = versionID
			design["version_number"] = versionNumber
			design["canonical_model"] = json.RawMessage(model)
			design["model_checksum"] = checksum
		}
		designs = append(designs, design)
	}
	if err := rows.Err(); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read project designs"})
		return
	}

	respond(w, http.StatusOK, map[string]any{"project": item, "designs": designs})
}
