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
)

type FloorPlanRoom struct {
        ID      string  `json:"id"`
        Name    string  `json:"name"`
        Type    string  `json:"type"`
        X       float64 `json:"x_m"`
        Y       float64 `json:"y_m"`
        Width   float64 `json:"width_m"`
        Depth   float64 `json:"depth_m"`
}

type FloorPlanWall struct {
        ID        string  `json:"id"`
        X1        float64 `json:"x1_m"`
        Y1        float64 `json:"y1_m"`
        X2        float64 `json:"x2_m"`
        Y2        float64 `json:"y2_m"`
        Thickness float64 `json:"thickness_m"`
}

type FloorPlanOpening struct {
        ID       string  `json:"id"`
        WallID   string  `json:"wall_id"`
        Offset   float64 `json:"offset_m"`
        Width    float64 `json:"width_m"`
        Height   float64 `json:"height_m"`
        Sill     float64 `json:"sill_height_m,omitempty"`
}

type FloorPlan struct {
        FloorNumber int                `json:"floor_number"`
        Name        string             `json:"name"`
        Rooms       []FloorPlanRoom    `json:"rooms"`
        Walls       []FloorPlanWall    `json:"walls"`
        Doors       []FloorPlanOpening `json:"doors"`
        Windows     []FloorPlanOpening `json:"windows"`
}

type floorPlansRequest struct {
        FloorPlans []FloorPlan `json:"floor_plans"`
}

func finitePositive(value float64) bool {
        return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func finiteNonNegative(value float64) bool {
        return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func validateFloorPlans(plans []FloorPlan, landWidth, landDepth float64, buildingFloors int) string {
        if plans == nil {
                return "floor_plans must be an array"
        }
        if len(plans) > buildingFloors {
                return "floor plan count exceeds the building floor count"
        }

        seenFloors := map[int]bool{}

        for _, plan := range plans {
                if plan.FloorNumber < 1 || plan.FloorNumber > buildingFloors {
                        return "floor number is outside the building floor range"
                }
                if seenFloors[plan.FloorNumber] {
                        return "duplicate floor number"
                }
                seenFloors[plan.FloorNumber] = true

                plan.Name = strings.TrimSpace(plan.Name)
                if len(plan.Name) == 0 || len(plan.Name) > 100 {
                        return "floor name is required and must not exceed 100 characters"
                }

                if len(plan.Rooms) > 100 || len(plan.Walls) > 500 ||
                        len(plan.Doors) > 200 || len(plan.Windows) > 200 {
                        return "floor plan contains too many elements"
                }

                if plan.Rooms == nil || plan.Walls == nil ||
                        plan.Doors == nil || plan.Windows == nil {
                        return "rooms, walls, doors and windows must be arrays"
                }

                roomIDs := map[string]bool{}
                for _, room := range plan.Rooms {
                        id := strings.TrimSpace(room.ID)
                        name := strings.TrimSpace(room.Name)
                        if id == "" || len(id) > 80 || roomIDs[id] {
                                return "room IDs must be unique and non-empty"
                        }
                        roomIDs[id] = true

                        if name == "" || len(name) > 100 {
                                return "room name is required and must not exceed 100 characters"
                        }
                        if len(room.Type) > 50 {
                                return "room type must not exceed 50 characters"
                        }
                        if !finiteNonNegative(room.X) || !finiteNonNegative(room.Y) ||
                                !finitePositive(room.Width) || !finitePositive(room.Depth) {
                                return "room coordinates and dimensions are invalid"
                        }
                        if room.X+room.Width > landWidth+1e-9 ||
                                room.Y+room.Depth > landDepth+1e-9 {
                                return "room exceeds the land boundary"
                        }
                }

                wallIDs := map[string]bool{}
                wallLengths := map[string]float64{}

                for _, wall := range plan.Walls {
                        id := strings.TrimSpace(wall.ID)
                        if id == "" || len(id) > 80 || wallIDs[id] {
                                return "wall IDs must be unique and non-empty"
                        }
                        wallIDs[id] = true

                        if !finiteNonNegative(wall.X1) || !finiteNonNegative(wall.Y1) ||
                                !finiteNonNegative(wall.X2) || !finiteNonNegative(wall.Y2) ||
                                !finitePositive(wall.Thickness) {
                                return "wall coordinates or thickness are invalid"
                        }

                        if wall.X1 > landWidth || wall.X2 > landWidth ||
                                wall.Y1 > landDepth || wall.Y2 > landDepth {
                                return "wall exceeds the land boundary"
                        }

                        length := math.Hypot(wall.X2-wall.X1, wall.Y2-wall.Y1)
                        if length <= 1e-9 {
                                return "wall must have a non-zero length"
                        }
                        wallLengths[id] = length
                }

                openingIDs := map[string]bool{}
                for _, openings := range [][]FloorPlanOpening{plan.Doors, plan.Windows} {
                        for _, opening := range openings {
                                id := strings.TrimSpace(opening.ID)
                                if id == "" || len(id) > 80 || openingIDs[id] {
                                        return "opening IDs must be unique within a floor"
                                }
                                openingIDs[id] = true

                                wallLength, exists := wallLengths[opening.WallID]
                                if !exists {
                                        return "door or window references an unknown wall"
                                }
                                if !finiteNonNegative(opening.Offset) ||
                                        !finitePositive(opening.Width) ||
                                        !finitePositive(opening.Height) ||
                                        !finiteNonNegative(opening.Sill) {
                                        return "door or window dimensions are invalid"
                                }
                                if opening.Offset+opening.Width > wallLength+1e-9 {
                                        return "door or window exceeds its wall segment"
                                }
                        }
                }
        }

        return ""
}

func (h *Handler) getFloorPlans(w http.ResponseWriter, r *http.Request, rawID string) {
        projectID, err := uuid.Parse(rawID)
        if err != nil {
                respond(w, http.StatusBadRequest, map[string]string{"error": "invalid project ID"})
                return
        }

        var modelJSON []byte
        var versionID uuid.UUID
        var versionNumber int
        var checksum string

        err = h.pool.QueryRow(r.Context(), `
                SELECT v.canonical_model, v.id, v.version_number, COALESCE(v.model_checksum, '')
                FROM core.designs d
                JOIN core.design_versions v
                  ON v.design_id = d.id
                 AND v.version_number = d.current_version
                WHERE d.project_id = $1
                ORDER BY d.created_at DESC, d.id DESC
                LIMIT 1
        `, projectID).Scan(&modelJSON, &versionID, &versionNumber, &checksum)

        if errors.Is(err, pgx.ErrNoRows) {
                respond(w, http.StatusNotFound, map[string]string{"error": "project design not found"})
                return
        }
        if err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read floor plans"})
                return
        }

        var model map[string]any
        if err := json.Unmarshal(modelJSON, &model); err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "invalid canonical model"})
                return
        }

        var plans json.RawMessage = json.RawMessage("[]")
        if building, ok := model["building"].(map[string]any); ok {
                if value, exists := building["floor_plans"]; exists {
                        encoded, encodeErr := json.Marshal(value)
                        if encodeErr != nil {
                                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not encode floor plans"})
                                return
                        }
                        plans = encoded
                }
        }

        respond(w, http.StatusOK, map[string]any{
                "project_id": projectID,
                "version_id": versionID,
                "version_number": versionNumber,
                "model_checksum": checksum,
                "floor_plans": plans,
        })
}

func (h *Handler) saveFloorPlans(w http.ResponseWriter, r *http.Request, rawID string) {
        projectID, err := uuid.Parse(rawID)
        if err != nil {
                respond(w, http.StatusBadRequest, map[string]string{"error": "invalid project ID"})
                return
        }

        r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
        decoder := json.NewDecoder(r.Body)
        decoder.DisallowUnknownFields()

        var req floorPlansRequest
        if err := decoder.Decode(&req); err != nil {
                respond(w, http.StatusBadRequest, map[string]string{"error": "invalid floor plan request"})
                return
        }
        if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
                respond(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON object"})
                return
        }

        ctx := r.Context()
        tx, err := h.pool.Begin(ctx)
        if err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not start floor plan transaction"})
                return
        }
        defer func() { _ = tx.Rollback(ctx) }()

        var lockedID uuid.UUID
        err = tx.QueryRow(ctx,
                `SELECT id FROM core.projects WHERE id = $1 FOR UPDATE`,
                projectID,
        ).Scan(&lockedID)

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

        err = tx.QueryRow(ctx, `
                SELECT d.id, d.current_version, v.canonical_model
                FROM core.designs d
                JOIN core.design_versions v
                  ON v.design_id = d.id
                 AND v.version_number = d.current_version
                WHERE d.project_id = $1
                ORDER BY d.created_at DESC, d.id DESC
                LIMIT 1
                FOR UPDATE OF d
        `, projectID).Scan(&designID, &currentVersion, &previousModelJSON)

        if errors.Is(err, pgx.ErrNoRows) {
                respond(w, http.StatusConflict, map[string]string{"error": "project has no editable design"})
                return
        }
        if err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read current design"})
                return
        }

        var model map[string]any
        if err := json.Unmarshal(previousModelJSON, &model); err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not read canonical model"})
                return
        }

        site, _ := model["site"].(map[string]any)
        land, _ := site["land"].(map[string]any)
        building, _ := model["building"].(map[string]any)

        landWidth, widthOK := land["width_m"].(float64)
        landDepth, depthOK := land["depth_m"].(float64)
        buildingFloors, floorsOK := building["floors"].(float64)

        if !widthOK || !depthOK || !floorsOK ||
                !finitePositive(landWidth) || !finitePositive(landDepth) ||
                buildingFloors < 1 || buildingFloors > 2 {
                respond(w, http.StatusConflict, map[string]string{"error": "project dimensions or floor count are invalid"})
                return
        }

        validationError := validateFloorPlans(req.FloorPlans, landWidth, landDepth, int(buildingFloors))
        if validationError != "" {
                respond(w, http.StatusBadRequest, map[string]string{"error": validationError})
                return
        }

        if currentVersion < 1 {
                respond(w, http.StatusConflict, map[string]string{"error": "current design version is invalid"})
                return
        }

        building["floor_plans"] = req.FloorPlans

        modelJSON, err := json.Marshal(model)
        if err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not encode canonical model"})
                return
        }

        sum := sha256.Sum256(modelJSON)
        checksum := hex.EncodeToString(sum[:])
        nextVersion := currentVersion + 1

        _, err = tx.Exec(ctx, `
                UPDATE core.designs
                SET current_version = $1, updated_at = NOW()
                WHERE id = $2
        `, nextVersion, designID)
        if err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not advance design version"})
                return
        }

        var versionID uuid.UUID
        err = tx.QueryRow(ctx, `
                INSERT INTO core.design_versions
                (design_id, version_number, canonical_model, model_checksum)
                VALUES ($1, $2, $3::jsonb, $4)
                RETURNING id
        `, designID, nextVersion, string(modelJSON), checksum).Scan(&versionID)
        if err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not save floor plan version"})
                return
        }

        if err := tx.Commit(ctx); err != nil {
                respond(w, http.StatusInternalServerError, map[string]string{"error": "could not commit floor plan version"})
                return
        }

        respond(w, http.StatusOK, map[string]any{
                "project_id": projectID,
                "design_id": designID,
                "version_id": versionID,
                "version_number": nextVersion,
                "model_checksum": checksum,
                "floor_plans": req.FloorPlans,
        })
}