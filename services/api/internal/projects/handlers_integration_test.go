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

	sum := sha256.Sum256(created.CanonicalModel)
	expectedChecksum := hex.EncodeToString(sum[:])

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

	if !strings.Contains(createRec.Header().Get("Content-Type"), "application/json") {
		t.Error("create response Content-Type is not JSON")
	}
}
