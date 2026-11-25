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
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	Location         string                `json:"location"`
	LandWidth        float64               `json:"land_width_m"`
	LandDepth        float64               `json:"land_depth_m"`
	Orientation      string                `json:"orientation"`
	Floors           int                   `json:"floors"`
	DesignStyleID    string                `json:"design_style_id"`
	DesignStyleName  string                `json:"design_style_name"`
	DesignStyleImage string                `json:"design_style_image"`
	Footprint        []FootprintPoint      `json:"footprint,omitempty"`
	Construction     *BuildingConstruction `json:"construction,omitempty"`
}

type FootprintPoint struct {
	X float64 `json:"x_m"`
	Y float64 `json:"y_m"`
}

type BuildingConstruction struct {
	WallHeight    float64 `json:"wall_height_m"`
	SlabThickness float64 `json:"slab_thickness_m"`
	FloorToFloor  float64 `json:"floor_to_floor_height_m"`
}

func footprintCross(ax, ay, bx, by, cx, cy float64) float64 {
	return (bx-ax)*(cy-ay) - (by-ay)*(cx-ax)
}

func footprintOnSegment(ax, ay, bx, by, px, py float64) bool {
	const eps = 1e-9
	return px >= math.Min(ax, bx)-eps &&
		px <= math.Max(ax, bx)+eps &&
		py >= math.Min(ay, by)-eps &&
		py <= math.Max(ay, by)+eps
}

func footprintSegmentsIntersect(
	ax, ay, bx, by, cx, cy, dx, dy float64,
) bool {
	const eps = 1e-9
	o1 := footprintCross(ax, ay, bx, by, cx, cy)
	o2 := footprintCross(ax, ay, bx, by, dx, dy)
	o3 := footprintCross(cx, cy, dx, dy, ax, ay)
	o4 := footprintCross(cx, cy, dx, dy, bx, by)

	if ((o1 > eps && o2 < -eps) || (o1 < -eps && o2 > eps)) &&
		((o3 > eps && o4 < -eps) || (o3 < -eps && o4 > eps)) {
		return true
	}

	if math.Abs(o1) <= eps && footprintOnSegment(ax, ay, bx, by, cx, cy) {
		return true
	}
	if math.Abs(o2) <= eps && footprintOnSegment(ax, ay, bx, by, dx, dy) {
		return true
	}
	if math.Abs(o3) <= eps && footprintOnSegment(cx, cy, dx, dy, ax, ay) {
		return true
	}
	if math.Abs(o4) <= eps && footprintOnSegment(cx, cy, dx, dy, bx, by) {
		return true
	}
	return false
}

func validateBuildingFootprint(points []FootprintPoint, landWidth, landDepth float64) string {
	if len(points) < 3 || len(points) > 100 {
		return "footprint must contain between 3 and 100 points"
	}

	area2 := 0.0
	for i, point := range points {
		if !finiteNonNegative(point.X) || !finiteNonNegative(point.Y) ||
			point.X > landWidth || point.Y > landDepth {
			return "footprint coordinates must be finite and inside the land boundary"
		}

		next := points[(i+1)%len(points)]
		if math.Hypot(next.X-point.X, next.Y-point.Y) <= 1e-9 {
			return "footprint contains a zero-length edge"
		}
		area2 += point.X*next.Y - next.X*point.Y
	}

	if math.Abs(area2) <= 1e-9 {
		return "footprint polygon area must be greater than zero"
	}

	n := len(points)
	for i := 0; i < n; i++ {
		a := points[i]
		b := points[(i+1)%n]

		for j := i + 1; j < n; j++ {
			// Adjacent edges share a legitimate endpoint.
			if j == i || j == (i+1)%n || (j+1)%n == i {
				continue
			}

			c := points[j]
			d := points[(j+1)%n]
			if footprintSegmentsIntersect(
				a.X, a.Y, b.X, b.Y,
				c.X, c.Y, d.X, d.Y,
			) {
				return "footprint polygon edges must not intersect"
			}
		}
	}

	return ""
}

func validateBuildingConstruction(value *BuildingConstruction, floors int) string {
	if value == nil {
		return ""
	}
	if !finitePositive(value.WallHeight) ||
		!finitePositive(value.SlabThickness) ||
		!finitePositive(value.FloorToFloor) {
		return "construction dimensions must be finite and greater than zero"
	}
	if floors > 1 && value.FloorToFloor < value.WallHeight {
		return "floor-to-floor height must not be less than wall height"
	}
	return ""
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
			respond(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
		}
		return
	}

	const projectPrefix = "/api/v1/projects/"
	if strings.HasPrefix(r.URL.Path, projectPrefix) {
		route := strings.TrimPrefix(r.URL.Path, projectPrefix)
		parts := strings.Split(route, "/")

		if len(parts) == 2 && parts[0] != "" && parts[1] == "floor-plans" {
			switch r.Method {
			case http.MethodGet:
				h.getFloorPlans(w, r, parts[0])
			case http.MethodPut:
				h.saveFloorPlans(w, r, parts[0])
			default:
				w.Header().Set("Allow", "GET, PUT")
				respond(w, http.StatusMethodNotAllowed, map[string]string{
					"error": "method not allowed",
				})
			}
			return
		}

		if len(parts) == 1 && parts[0] != "" {
			switch r.Method {
			case http.MethodGet:
				h.get(w, r, parts[0])
			case http.MethodPut:
				h.update(w, r, parts[0])
			default:
				w.Header().Set("Allow", "GET, PUT")
				respond(w, http.StatusMethodNotAllowed, map[string]string{
					"error": "method not allowed",
				})
			}
			return
		}
	}

	respond(w, http.StatusNotFound, map[string]string{
		"error": "not found",
	})
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

	req.DesignStyleID = strings.TrimSpace(req.DesignStyleID)
	req.DesignStyleName = strings.TrimSpace(req.DesignStyleName)
	req.DesignStyleImage = strings.TrimSpace(req.DesignStyleImage)

	hasStyle := req.DesignStyleID != "" ||
		req.DesignStyleName != "" || req.DesignStyleImage != ""

	if hasStyle {
		expected, exists := houseStyleCatalog[req.DesignStyleID]
		if !exists ||
			req.DesignStyleName != expected.Name ||
			req.DesignStyleImage != expected.Image {
			respond(w, http.StatusBadRequest, map[string]string{
				"error": "invalid or inconsistent design style",
			})
			return
		}
	}

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

	if hasStyle {
		building := model["building"].(map[string]any)
		building["style"] = map[string]any{
			"id":    req.DesignStyleID,
			"name":  req.DesignStyleName,
			"image": req.DesignStyleImage,
		}
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

// update creates a new immutable design version for an existing project.
func (h *Handler) update(w http.ResponseWriter, r *http.Request, rawID string) {
	projectID, err := uuid.Parse(rawID)
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid project ID"})
		return
	}

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
	req.DesignStyleID = strings.TrimSpace(req.DesignStyleID)
	req.DesignStyleName = strings.TrimSpace(req.DesignStyleName)
	req.DesignStyleImage = strings.TrimSpace(req.DesignStyleImage)

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
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid land orientation"})
		return
	}

	if req.Footprint != nil {
		if message := validateBuildingFootprint(req.Footprint, req.LandWidth, req.LandDepth); message != "" {
			respond(w, http.StatusBadRequest, map[string]string{"error": message})
			return
		}
	}

	if message := validateBuildingConstruction(req.Construction, req.Floors); message != "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": message})
		return
	}
	hasStyle := req.DesignStyleID != "" ||
		req.DesignStyleName != "" ||
		req.DesignStyleImage != ""

	if hasStyle {
		expected, exists := houseStyleCatalog[req.DesignStyleID]
		if !exists ||
			req.DesignStyleName != expected.Name ||
			req.DesignStyleImage != expected.Image {
			respond(w, http.StatusBadRequest, map[string]string{"error": "invalid or inconsistent design style"})
			return
		}
	}

	ctx := r.Context()

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not start database transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var existingProjectID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM core.projects WHERE id = $1 FOR UPDATE`,
		projectID,
	).Scan(&existingProjectID)

	if errors.Is(err, pgx.ErrNoRows) {
		respond(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not lock project"})
		return
	}

	var designID uuid.UUID
	var currentVersion int
	var previousModelJSON []byte

	err = tx.QueryRow(ctx,
		`SELECT d.id, d.current_version, v.canonical_model
         FROM core.designs d
         JOIN core.design_versions v
           ON v.design_id = d.id
          AND v.version_number = d.current_version
         WHERE d.project_id = $1
         ORDER BY d.created_at DESC, d.id DESC
         LIMIT 1
         FOR UPDATE OF d`,
		projectID,
	).Scan(&designID, &currentVersion, &previousModelJSON)

	if errors.Is(err, pgx.ErrNoRows) {
		respond(w, http.StatusConflict, map[string]string{"error": "project has no editable design"})
		return
	}
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read current design"})
		return
	}

	// Preserve the existing style when the edit request does not specify one.
	if !hasStyle {
		var previousModel map[string]any
		if err := json.Unmarshal(previousModelJSON, &previousModel); err != nil {
			respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read previous design model"})
			return
		}

		if previousBuilding, ok := previousModel["building"].(map[string]any); ok {
			if previousStyle, exists := previousBuilding["style"]; exists {
				if styleMap, ok := previousStyle.(map[string]any); ok {
					styleID, _ := styleMap["id"].(string)
					styleName, _ := styleMap["name"].(string)
					styleImage, _ := styleMap["image"].(string)

					expected, valid := houseStyleCatalog[styleID]
					if !valid || styleName != expected.Name || styleImage != expected.Image {
						respond(w, http.StatusConflict, map[string]string{"error": "existing design style is invalid"})
						return
					}

					req.DesignStyleID = styleID
					req.DesignStyleName = styleName
					req.DesignStyleImage = styleImage
					hasStyle = true
				}
			}
		}
	}

	if currentVersion < 1 {
		respond(w, http.StatusConflict, map[string]string{"error": "current design version is invalid"})
		return
	}

	area := math.Round(req.LandWidth*req.LandDepth*100) / 100

	// Preserve existing model properties not edited by this request.
	model := map[string]any{}
	if err := json.Unmarshal(previousModelJSON, &model); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{
			"error": "could not read previous design model",
		})
		return
	}

	if model["schema_version"] == nil {
		model["schema_version"] = "1.0"
	}

	projectModel, ok := model["project"].(map[string]any)
	if !ok {
		projectModel = map[string]any{}
		model["project"] = projectModel
	}
	projectModel["name"] = req.Name
	projectModel["description"] = req.Description

	siteModel, ok := model["site"].(map[string]any)
	if !ok {
		siteModel = map[string]any{}
		model["site"] = siteModel
	}
	siteModel["location"] = req.Location

	landModel, ok := siteModel["land"].(map[string]any)
	if !ok {
		landModel = map[string]any{}
		siteModel["land"] = landModel
	}
	if landModel["shape"] == nil {
		landModel["shape"] = "rectangle"
	}
	landModel["width_m"] = req.LandWidth
	landModel["depth_m"] = req.LandDepth
	landModel["area_m2"] = area
	landModel["orientation"] = req.Orientation

	buildingModel, ok := model["building"].(map[string]any)
	if !ok {
		buildingModel = map[string]any{}
		model["building"] = buildingModel
	}
	buildingModel["floors"] = req.Floors

	// Update optional 3D geometry only when supplied.
	if req.Footprint != nil {
		footprintModel := make([]map[string]any, 0, len(req.Footprint))
		for _, point := range req.Footprint {
			footprintModel = append(footprintModel, map[string]any{
				"x_m": point.X,
				"y_m": point.Y,
			})
		}
		buildingModel["footprint"] = footprintModel
	}

	if req.Construction != nil {
		buildingModel["construction"] = map[string]any{
			"wall_height_m":           req.Construction.WallHeight,
			"slab_thickness_m":        req.Construction.SlabThickness,
			"floor_to_floor_height_m": req.Construction.FloorToFloor,
		}
	}
	if hasStyle {
		buildingModel["style"] = map[string]any{
			"id":    req.DesignStyleID,
			"name":  req.DesignStyleName,
			"image": req.DesignStyleImage,
		}
	}

	unitsModel, ok := model["units"].(map[string]any)
	if !ok {
		unitsModel = map[string]any{}
		model["units"] = unitsModel
	}
	if unitsModel["length"] == nil {
		unitsModel["length"] = "m"
	}
	if unitsModel["area"] == nil {
		unitsModel["area"] = "m2"
	}

	modelJSON, err := json.Marshal(model)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not encode design model"})
		return
	}

	sum := sha256.Sum256(modelJSON)
	checksum := hex.EncodeToString(sum[:])
	nextVersion := currentVersion + 1

	_, err = tx.Exec(ctx,
		`UPDATE core.projects
         SET name = $1, description = $2, updated_at = NOW()
         WHERE id = $3`,
		req.Name, req.Description, projectID,
	)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not update project"})
		return
	}

	_, err = tx.Exec(ctx,
		`UPDATE core.designs
         SET name = $1, current_version = $2, updated_at = NOW()
         WHERE id = $3`,
		req.Name+" - Design", nextVersion, designID,
	)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not update design"})
		return
	}

	var versionID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO core.design_versions
         (design_id, version_number, canonical_model, model_checksum)
         VALUES ($1, $2, $3::jsonb, $4)
         RETURNING id`,
		designID, nextVersion, string(modelJSON), checksum,
	).Scan(&versionID)

	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not save new design version"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "could not commit project update"})
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"project_id":      projectID,
		"design_id":       designID,
		"version_id":      versionID,
		"version_number":  nextVersion,
		"model_checksum":  checksum,
		"canonical_model": json.RawMessage(modelJSON),
	})
}
