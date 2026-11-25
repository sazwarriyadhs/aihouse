package projects

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ai-house-generator/api/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

const integrationDatabaseName = "ai_house_generator_test"

func TestProjectCRUDIntegration(t *testing.T) {
	if os.Getenv("AHG_INTEGRATION") != "1" {
		t.Skip("set AHG_INTEGRATION=1 to run isolated database integration test")
	}

	if err := godotenv.Load("../../../../.env"); err != nil {
		t.Fatalf("load repository .env: %v", err)
	}

	cfg, err := database.LoadRuntimeConfig()
	if err != nil {
		t.Fatalf("load database config: %v", err)
	}

	cfg.Database = integrationDatabaseName
	if cfg.Database != integrationDatabaseName {
		t.Fatal("refusing to run integration test against an unexpected database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("connect isolated test database: %v", err)
	}
	defer pool.Close()

	var currentDatabase string
	if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&currentDatabase); err != nil {
		t.Fatalf("verify database identity: %v", err)
	}
	if currentDatabase != integrationDatabaseName {
		t.Fatalf("unsafe database target %q", currentDatabase)
	}

	handler := NewHandler(pool)
	projectName := "Integration Test " + uuid.NewString()

	requestBody, err := json.Marshal(createRequest{
		Name:        projectName,
		Description: "Automated CRUD integration test",
		Location:    "Bogor",
		LandWidth:   10,
		LandDepth:   15,
		Orientation: "N",
		Floors:      2,
	})
	if err != nil {
		t.Fatal(err)
	}

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects",
		bytes.NewReader(requestBody),
	)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body: %s",
			createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	var created struct {
		ProjectID      uuid.UUID       `json:"project_id"`
		DesignID       uuid.UUID       `json:"design_id"`
		VersionID      uuid.UUID       `json:"version_id"`
		CanonicalModel json.RawMessage `json:"canonical_model"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ProjectID == uuid.Nil || created.DesignID == uuid.Nil || created.VersionID == uuid.Nil {
		t.Fatal("create response contains an empty ID")
	}

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		_, cleanupErr := pool.Exec(
			cleanupCtx,
			"DELETE FROM core.projects WHERE id = $1",
			created.ProjectID,
		)
		if cleanupErr != nil {
			t.Errorf("cleanup integration project: %v", cleanupErr)
		}
	}()

	// Verify canonical model and calculated land area.
	var model struct {
		Site struct {
			Land struct {
				Width       float64 `json:"width_m"`
				Depth       float64 `json:"depth_m"`
				Area        float64 `json:"area_m2"`
				Orientation string  `json:"orientation"`
			} `json:"land"`
		} `json:"site"`
		Building struct {
			Floors int `json:"floors"`
		} `json:"building"`
	}
	if err := json.Unmarshal(created.CanonicalModel, &model); err != nil {
		t.Fatalf("decode canonical model: %v", err)
	}
	if model.Site.Land.Area != 150 {
		t.Errorf("land area = %v, want 150", model.Site.Land.Area)
	}
	if model.Site.Land.Width != 10 || model.Site.Land.Depth != 15 {
		t.Errorf("land dimensions = %v x %v, want 10 x 15",
			model.Site.Land.Width, model.Site.Land.Depth)
	}
	if model.Site.Land.Orientation != "N" {
		t.Errorf("orientation = %q, want N", model.Site.Land.Orientation)
	}
	if model.Building.Floors != 2 {
		t.Errorf("floors = %d, want 2", model.Building.Floors)
	}

	// Inject future floor-planning data into version 1.
	floorPlansJSON := []byte(`[{
                "floor_number": 1,
                "name": "Ground Floor",
                "rooms": [{
                        "id": "room-regression-1",
                        "name": "Living Room",
                        "type": "living",
                        "x_m": 1,
                        "y_m": 2,
                        "width_m": 4,
                        "depth_m": 5
                }]
        }]`)

	var fixtureModelJSON []byte
	err = pool.QueryRow(ctx, `
                UPDATE core.design_versions
                SET canonical_model = jsonb_set(
                        canonical_model,
                        '{building,floor_plans}',
                        $2::jsonb,
                        true
                )
                WHERE id = $1
                RETURNING canonical_model
        `, created.VersionID, floorPlansJSON).Scan(&fixtureModelJSON)
	if err != nil {
		t.Fatalf("inject floor planning fixture: %v", err)
	}

	fixtureSum := sha256.Sum256(fixtureModelJSON)
	expectedChecksum := hex.EncodeToString(fixtureSum[:])

	_, err = pool.Exec(ctx, `
                UPDATE core.design_versions
                SET model_checksum = $2
                WHERE id = $1
        `, created.VersionID, expectedChecksum)
	if err != nil {
		t.Fatalf("update fixture checksum: %v", err)
	}

	// Verify project appears in collection.
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body: %s", listRec.Code, listRec.Body.String())
	}

	var listResponse struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	found := false
	for _, item := range listResponse.Items {
		if item.ID == created.ProjectID.String() && item.Name == projectName {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("created project not found in project collection")
	}

	// Verify project detail and immutable initial version metadata.
	detailPath := fmt.Sprintf("/api/v1/projects/%s", created.ProjectID)
	detailReq := httptest.NewRequest(http.MethodGet, detailPath, nil)
	detailRec := httptest.NewRecorder()
	handler.ServeHTTP(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body: %s", detailRec.Code, detailRec.Body.String())
	}

	var detail struct {
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
		Designs []struct {
			ID             string `json:"id"`
			CurrentVersion int    `json:"current_version"`
			VersionID      string `json:"version_id"`
			VersionNumber  int    `json:"version_number"`
			ModelChecksum  string `json:"model_checksum"`
		} `json:"designs"`
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}

	if detail.Project.ID != created.ProjectID.String() {
		t.Errorf("detail project ID = %q, want %q", detail.Project.ID, created.ProjectID)
	}
	if detail.Project.Name != projectName {
		t.Errorf("detail project name = %q, want %q", detail.Project.Name, projectName)
	}
	if len(detail.Designs) != 1 {
		t.Fatalf("design count = %d, want 1", len(detail.Designs))
	}

	design := detail.Designs[0]
	if design.ID != created.DesignID.String() {
		t.Errorf("design ID = %q, want %q", design.ID, created.DesignID)
	}
	if design.CurrentVersion != 1 || design.VersionNumber != 1 {
		t.Errorf("version metadata = current %d, version %d; want 1 and 1",
			design.CurrentVersion, design.VersionNumber)
	}
	if design.VersionID != created.VersionID.String() {
		t.Errorf("version ID = %q, want %q", design.VersionID, created.VersionID)
	}
	if design.ModelChecksum != expectedChecksum {
		t.Errorf("checksum = %q, want %q", design.ModelChecksum, expectedChecksum)
	}

	// Verify all three records were persisted in the isolated database.
	var projectCount, designCount, versionCount int
	err = pool.QueryRow(ctx, `
        SELECT
            (SELECT COUNT(*) FROM core.projects WHERE id = $1),
            (SELECT COUNT(*) FROM core.designs WHERE id = $2),
            (SELECT COUNT(*) FROM core.design_versions WHERE id = $3)
    `, created.ProjectID, created.DesignID, created.VersionID).
		Scan(&projectCount, &designCount, &versionCount)
	if err != nil {
		t.Fatalf("verify persisted records: %v", err)
	}
	if projectCount != 1 || designCount != 1 || versionCount != 1 {
		t.Errorf("persisted counts = %d, %d, %d; want 1, 1, 1",
			projectCount, designCount, versionCount)
	}

	// Edit the project and create an immutable second design version.
	updatedProjectName := projectName + " Updated"
	updateBody, err := json.Marshal(createRequest{
		Name:        updatedProjectName,
		Description: "Updated integration test",
		Location:    "Jakarta",
		LandWidth:   12,
		LandDepth:   20,
		Orientation: "E",
		Floors:      1,
	})
	if err != nil {
		t.Fatalf("encode update request: %v", err)
	}

	updateReq := httptest.NewRequest(
		http.MethodPut,
		detailPath,
		bytes.NewReader(updateBody),
	)
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body: %s",
			updateRec.Code, http.StatusOK, updateRec.Body.String())
	}

	var updated struct {
		ProjectID      string          `json:"project_id"`
		DesignID       string          `json:"design_id"`
		VersionID      string          `json:"version_id"`
		VersionNumber  int             `json:"version_number"`
		ModelChecksum  string          `json:"model_checksum"`
		CanonicalModel json.RawMessage `json:"canonical_model"`
	}
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}

	if updated.ProjectID != created.ProjectID.String() {
		t.Errorf("updated project ID = %q, want %q", updated.ProjectID, created.ProjectID)
	}
	if updated.DesignID != created.DesignID.String() {
		t.Errorf("updated design ID = %q, want %q", updated.DesignID, created.DesignID)
	}
	if updated.VersionID == "" || updated.VersionID == created.VersionID.String() {
		t.Errorf("updated version ID = %q; expected a new version ID", updated.VersionID)
	}
	if updated.VersionNumber != 2 {
		t.Errorf("updated version number = %d, want 2", updated.VersionNumber)
	}

	// Verify future floor-planning data survives project information updates.
	var preservedModel struct {
		Building struct {
			FloorPlans []struct {
				FloorNumber int `json:"floor_number"`
				Rooms       []struct {
					ID    string  `json:"id"`
					Name  string  `json:"name"`
					X     float64 `json:"x_m"`
					Y     float64 `json:"y_m"`
					Width float64 `json:"width_m"`
					Depth float64 `json:"depth_m"`
				} `json:"rooms"`
			} `json:"floor_plans"`
		} `json:"building"`
	}

	if err := json.Unmarshal(updated.CanonicalModel, &preservedModel); err != nil {
		t.Fatalf("decode preserved floor planning model: %v", err)
	}

	if len(preservedModel.Building.FloorPlans) != 1 {
		t.Fatalf("floor plan count = %d, want 1",
			len(preservedModel.Building.FloorPlans))
	}

	floorPlan := preservedModel.Building.FloorPlans[0]
	if floorPlan.FloorNumber != 1 {
		t.Errorf("preserved floor number = %d, want 1", floorPlan.FloorNumber)
	}

	if len(floorPlan.Rooms) != 1 {
		t.Fatalf("preserved room count = %d, want 1", len(floorPlan.Rooms))
	}

	room := floorPlan.Rooms[0]
	if room.ID != "room-regression-1" ||
		room.Name != "Living Room" ||
		room.X != 1 ||
		room.Y != 2 ||
		room.Width != 4 ||
		room.Depth != 5 {
		t.Errorf("floor planning room was not preserved: %+v", room)
	}
	updatedSum := sha256.Sum256(updated.CanonicalModel)
	updatedExpectedChecksum := hex.EncodeToString(updatedSum[:])
	if updated.ModelChecksum != updatedExpectedChecksum {
		t.Errorf("updated checksum = %q, want %q",
			updated.ModelChecksum, updatedExpectedChecksum)
	}

	// Verify the original snapshot remains unchanged and version 2 is current.
	var storedProjectName string
	var storedCurrentVersion, storedVersionCount int
	var storedV1Checksum, storedV2Checksum string

	err = pool.QueryRow(ctx, `
      SELECT
          p.name,
          d.current_version,
          (SELECT COUNT(*) FROM core.design_versions WHERE design_id = d.id),
          (SELECT model_checksum FROM core.design_versions
             WHERE design_id = d.id AND version_number = 1),
          (SELECT model_checksum FROM core.design_versions
             WHERE design_id = d.id AND version_number = 2)
      FROM core.projects p
      JOIN core.designs d ON d.project_id = p.id
      WHERE p.id = $1 AND d.id = $2
  `, created.ProjectID, created.DesignID).Scan(
		&storedProjectName,
		&storedCurrentVersion,
		&storedVersionCount,
		&storedV1Checksum,
		&storedV2Checksum,
	)
	if err != nil {
		t.Fatalf("verify version history: %v", err)
	}

	if storedProjectName != updatedProjectName {
		t.Errorf("stored project name = %q, want %q", storedProjectName, updatedProjectName)
	}
	if storedCurrentVersion != 2 {
		t.Errorf("stored current version = %d, want 2", storedCurrentVersion)
	}
	if storedVersionCount != 2 {
		t.Errorf("stored version count = %d, want 2", storedVersionCount)
	}
	if storedV1Checksum != expectedChecksum {
		t.Errorf("version 1 checksum changed: got %q, want %q",
			storedV1Checksum, expectedChecksum)
	}
	if storedV2Checksum != updatedExpectedChecksum {
		t.Errorf("version 2 checksum = %q, want %q",
			storedV2Checksum, updatedExpectedChecksum)
	}

	// Verify GET detail returns the new active version.
	updatedDetailReq := httptest.NewRequest(http.MethodGet, detailPath, nil)
	updatedDetailRec := httptest.NewRecorder()
	handler.ServeHTTP(updatedDetailRec, updatedDetailReq)

	if updatedDetailRec.Code != http.StatusOK {
		t.Fatalf("updated detail status = %d; body: %s",
			updatedDetailRec.Code, updatedDetailRec.Body.String())
	}

	if err := json.Unmarshal(updatedDetailRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode updated detail response: %v", err)
	}
	if len(detail.Designs) != 1 {
		t.Fatalf("updated detail design count = %d, want 1", len(detail.Designs))
	}

	activeDesign := detail.Designs[0]
	if activeDesign.CurrentVersion != 2 || activeDesign.VersionNumber != 2 {
		t.Errorf("active version metadata = current %d, version %d; want 2 and 2",
			activeDesign.CurrentVersion, activeDesign.VersionNumber)
	}
	if activeDesign.VersionID != updated.VersionID {
		t.Errorf("active version ID = %q, want %q",
			activeDesign.VersionID, updated.VersionID)
	}
	if activeDesign.ModelChecksum != updatedExpectedChecksum {
		t.Errorf("active checksum = %q, want %q",
			activeDesign.ModelChecksum, updatedExpectedChecksum)
	}
	if !strings.Contains(createRec.Header().Get("Content-Type"), "application/json") {
		t.Error("create response Content-Type is not JSON")
	}
}
