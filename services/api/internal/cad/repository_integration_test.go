package cad

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ai-house-generator/api/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

const cadIntegrationDatabase = "ai_house_generator_test"

func TestCADRepositoryIntegration(t *testing.T) {
	if os.Getenv("AHG_INTEGRATION") != "1" {
		t.Skip("set AHG_INTEGRATION=1 to run isolated CAD integration test")
	}

	if err := godotenv.Load("../../../../.env"); err != nil {
		t.Fatalf("load repository .env: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	migratorConfig, err := database.LoadMigratorConfig()
	if err != nil {
		t.Fatalf("load migrator config: %v", err)
	}
	migratorConfig.Database = cadIntegrationDatabase

	migratorPool, err := database.Connect(ctx, migratorConfig)
	if err != nil {
		t.Fatalf("connect test migrator database: %v", err)
	}
	defer migratorPool.Close()

	var databaseName string
	if err := migratorPool.QueryRow(
		ctx, "SELECT current_database()",
	).Scan(&databaseName); err != nil {
		t.Fatalf("verify migrator database: %v", err)
	}
	if databaseName != cadIntegrationDatabase {
		t.Fatalf("unsafe migrator database %q", databaseName)
	}

	// Apply embedded migrations only to the isolated test database.
	if err := database.RunMigrations(ctx, migratorPool); err != nil {
		t.Fatalf("run test database migrations: %v", err)
	}

	runtimeConfig, err := database.LoadRuntimeConfig()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	runtimeConfig.Database = cadIntegrationDatabase

	pool, err := database.Connect(ctx, runtimeConfig)
	if err != nil {
		t.Fatalf("connect test runtime database: %v", err)
	}
	defer pool.Close()

	if err := pool.QueryRow(
		ctx, "SELECT current_database()",
	).Scan(&databaseName); err != nil {
		t.Fatalf("verify runtime database: %v", err)
	}
	if databaseName != cadIntegrationDatabase {
		t.Fatalf("unsafe runtime database %q", databaseName)
	}

	projectID := uuid.New()
	designID := uuid.New()
	versionID := uuid.New()
	jobID := uuid.New()
	executorJobID := uuid.New()

	// Create isolated fixture using runtime permissions.
	_, err = pool.Exec(ctx, `
        INSERT INTO core.projects (id, name, status)
        VALUES ($1, $2, 'draft')
    `, projectID, "CAD Integration "+projectID.String())
	if err != nil {
		t.Fatalf("insert test project: %v", err)
	}

	// Cleanup uses the migrator role because runtime cannot delete artifacts.
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cleanupCancel()

		conn, acquireErr := migratorPool.Acquire(cleanupCtx)
		if acquireErr != nil {
			t.Errorf("acquire cleanup connection: %v", acquireErr)
			return
		}
		defer conn.Release()

		if _, roleErr := conn.Exec(cleanupCtx, "SET ROLE ahg_owner"); roleErr != nil {
			t.Errorf("set cleanup role: %v", roleErr)
			return
		}

		if _, deleteErr := conn.Exec(
			cleanupCtx,
			"DELETE FROM core.cad_artifacts WHERE project_id = $1",
			projectID,
		); deleteErr != nil {
			t.Errorf("cleanup CAD artifacts: %v", deleteErr)
		}

		if _, deleteErr := conn.Exec(
			cleanupCtx,
			"DELETE FROM core.projects WHERE id = $1",
			projectID,
		); deleteErr != nil {
			t.Errorf("cleanup CAD project: %v", deleteErr)
		}

		if _, resetErr := conn.Exec(cleanupCtx, "RESET ROLE"); resetErr != nil {
			t.Errorf("reset cleanup role: %v", resetErr)
		}
	}()

	_, err = pool.Exec(ctx, `
        INSERT INTO core.designs (
            id, project_id, name, current_version, status
        )
        VALUES ($1, $2, 'CAD Integration Design', 1, 'draft')
    `, designID, projectID)
	if err != nil {
		t.Fatalf("insert test design: %v", err)
	}

	canonicalModel := testCADModel()
	modelHash := sha256.Sum256(canonicalModel)
	expectedModelChecksum := hex.EncodeToString(modelHash[:])

	_, err = pool.Exec(ctx, `
        INSERT INTO core.design_versions (
            id, design_id, version_number,
            canonical_model, model_checksum
        )
        VALUES ($1, $2, 1, $3::jsonb, $4)
    `, versionID, designID, canonicalModel, expectedModelChecksum)
	if err != nil {
		t.Fatalf("insert test design version: %v", err)
	}

	repository := NewRepository(pool)

	t.Run("load immutable design version", func(t *testing.T) {
		version, loadErr := repository.LoadVersion(
			ctx,
			projectID.String(),
			designID.String(),
			versionID.String(),
		)
		if loadErr != nil {
			t.Fatalf("load version: %v", loadErr)
		}

		if version.ProjectID != projectID.String() {
			t.Errorf("project ID = %s", version.ProjectID)
		}
		if version.DesignID != designID.String() {
			t.Errorf("design ID = %s", version.DesignID)
		}
		if version.VersionID != versionID.String() {
			t.Errorf("version ID = %s", version.VersionID)
		}
		if version.VersionNumber != 1 {
			t.Errorf("version number = %d, want 1", version.VersionNumber)
		}
		if version.ModelChecksum != expectedModelChecksum {
			t.Errorf("model checksum mismatch")
		}
		if len(version.CanonicalModel) == 0 {
			t.Fatal("canonical model is empty")
		}
	})

	t.Run("reject mismatched project", func(t *testing.T) {
		_, loadErr := repository.LoadVersion(
			ctx,
			uuid.NewString(),
			designID.String(),
			versionID.String(),
		)
		if loadErr == nil {
			t.Fatal("expected mismatched project to be rejected")
		}
	})

	artifactInput := ArtifactInput{
		JobID:      jobID.String(),
		ProjectID:  projectID.String(),
		DesignID:   designID.String(),
		VersionID:  versionID.String(),
		Dimension:  "2d",
		Format:     "dxf",
		StorageKey: "cad/" + jobID.String() + "/2d/model.dxf",
		Checksum:   strings.Repeat("b", 64),
		SizeBytes:  1234,
	}

	t.Run("create artifact metadata", func(t *testing.T) {
		artifact, createErr := repository.CreateArtifact(ctx, artifactInput)
		if createErr != nil {
			t.Fatalf("create artifact: %v", createErr)
		}

		if artifact.ID == "" {
			t.Fatal("artifact ID is empty")
		}
		if artifact.JobID != jobID.String() {
			t.Errorf("job ID = %s", artifact.JobID)
		}
		if artifact.StorageKey != artifactInput.StorageKey {
			t.Errorf("storage key mismatch")
		}
		if artifact.Checksum != artifactInput.Checksum {
			t.Errorf("checksum mismatch")
		}
		if artifact.SizeBytes != artifactInput.SizeBytes {
			t.Errorf("size = %d", artifact.SizeBytes)
		}
	})

	t.Run("idempotent artifact creation", func(t *testing.T) {
		first, createErr := repository.CreateArtifact(ctx, artifactInput)
		if createErr != nil {
			t.Fatalf("first creation: %v", createErr)
		}

		second, createErr := repository.CreateArtifact(ctx, artifactInput)
		if createErr != nil {
			t.Fatalf("repeated creation: %v", createErr)
		}

		if first.ID != second.ID {
			t.Fatalf("artifact IDs differ: %s and %s", first.ID, second.ID)
		}
	})

	t.Run("reject conflicting artifact metadata", func(t *testing.T) {
		conflicting := artifactInput
		conflicting.Checksum = strings.Repeat("c", 64)

		_, createErr := repository.CreateArtifact(ctx, conflicting)
		if createErr == nil {
			t.Fatal("expected conflicting metadata to be rejected")
		}
	})

	t.Run("reject inconsistent project design version chain", func(t *testing.T) {
		inconsistent := artifactInput
		inconsistent.JobID = uuid.NewString()
		inconsistent.ProjectID = uuid.NewString()
		inconsistent.StorageKey = "cad/" + inconsistent.JobID + "/2d/model.dxf"

		_, createErr := repository.CreateArtifact(ctx, inconsistent)
		if createErr == nil {
			t.Fatal("expected inconsistent project/design/version chain to be rejected")
		}
	})

	t.Run("batch artifact creation is atomic", func(t *testing.T) {
		first := artifactInput
		first.Format = "fcstd"
		first.StorageKey = "cad/" + jobID.String() + "/2d/model.fcstd"
		first.Checksum = strings.Repeat("d", 64)
		first.SizeBytes = 2345

		second := artifactInput
		second.Dimension = "3d"
		second.Format = "step"
		second.StorageKey = "cad/" + jobID.String() + "/3d/model.step"
		second.Checksum = strings.Repeat("e", 64)
		second.SizeBytes = 3456

		created, batchErr := repository.CreateArtifacts(ctx, []ArtifactInput{first, second})
		if batchErr != nil {
			t.Fatalf("create artifact batch: %v", batchErr)
		}
		if len(created) != 2 {
			t.Fatalf("created %d artifacts, want 2", len(created))
		}

		repeated, batchErr := repository.CreateArtifacts(ctx, []ArtifactInput{first, second})
		if batchErr != nil {
			t.Fatalf("repeat artifact batch: %v", batchErr)
		}
		if len(repeated) != 2 || repeated[0].ID != created[0].ID || repeated[1].ID != created[1].ID {
			t.Fatal("repeated batch did not return the original artifact rows")
		}
	})

	t.Run("batch conflict rolls back earlier inserts", func(t *testing.T) {
		newArtifact := artifactInput
		newArtifact.Dimension = "3d"
		newArtifact.Format = "fcstd"
		newArtifact.StorageKey = "cad/" + jobID.String() + "/3d/model.fcstd"
		newArtifact.Checksum = strings.Repeat("f", 64)
		newArtifact.SizeBytes = 4567

		conflict := artifactInput
		conflict.Checksum = strings.Repeat("c", 64)

		_, batchErr := repository.CreateArtifacts(ctx, []ArtifactInput{
			newArtifact,
			conflict,
		})
		if batchErr == nil {
			t.Fatal("expected batch conflict")
		}

		var count int
		queryErr := pool.QueryRow(ctx, `
            SELECT count(*)
            FROM core.cad_artifacts
            WHERE job_id = $1::uuid
              AND dimension = '3d'
              AND format = 'fcstd'
        `, jobID).Scan(&count)
		if queryErr != nil {
			t.Fatalf("verify batch rollback: %v", queryErr)
		}
		if count != 0 {
			t.Fatalf("rollback left %d partial artifact rows", count)
		}
	})

	t.Run("load artifacts by job", func(t *testing.T) {
		artifacts, loadErr := repository.LoadArtifactsByJob(ctx, jobID.String())
		if loadErr != nil {
			t.Fatalf("load artifacts by job: %v", loadErr)
		}
		if len(artifacts) != 3 {
			t.Fatalf("loaded %d artifacts, want 3", len(artifacts))
		}
	})

	t.Run("FreeCAD exports both dimensions and persists idempotently", func(t *testing.T) {
		script, readErr := os.ReadFile("freecad_export.py")
		if readErr != nil {
			t.Fatalf("read FreeCAD exporter: %v", readErr)
		}

		const pythonPath = `D:\freecad\bin\python.exe`
		if _, statErr := os.Stat(pythonPath); statErr != nil {
			t.Fatalf("FreeCAD Python unavailable: %v", statErr)
		}

		storageRoot := t.TempDir()
		executor, executorErr := NewExecutor(repository, storageRoot, pythonPath, script)
		if executorErr != nil {
			t.Fatalf("create CAD executor: %v", executorErr)
		}

		request := ExportRequest{
			ProjectID: projectID.String(),
			DesignID:  designID.String(),
			VersionID: versionID.String(),
			Target:    "both",
		}
		payload, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			t.Fatalf("marshal export request: %v", marshalErr)
		}

		executorCtx, executorCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer executorCancel()

		if processErr := executor.Process(executorCtx, executorJobID.String(), "cad.export", string(payload)); processErr != nil {
			t.Fatalf("execute FreeCAD export: %v", processErr)
		}

		artifacts, loadErr := repository.LoadArtifactsByJob(executorCtx, executorJobID.String())
		if loadErr != nil {
			t.Fatalf("load exported artifacts: %v", loadErr)
		}

		expected := map[string]bool{
			"2d/dxf":   true,
			"2d/fcstd": true,
			"3d/fcstd": true,
			"3d/step":  true,
		}
		if len(artifacts) != len(expected) {
			t.Fatalf("got %d artifacts, want %d", len(artifacts), len(expected))
		}

		firstRun := make(map[string]Artifact, len(artifacts))
		for _, artifact := range artifacts {
			key := artifact.Dimension + "/" + artifact.Format
			if !expected[key] {
				t.Errorf("unexpected artifact type: %s", key)
				continue
			}
			if _, exists := firstRun[key]; exists {
				t.Errorf("duplicate artifact type: %s", key)
				continue
			}
			firstRun[key] = artifact

			filePath := filepath.Join(storageRoot, filepath.FromSlash(artifact.StorageKey))
			info, statErr := os.Stat(filePath)
			if statErr != nil {
				t.Errorf("%s file missing: %v", key, statErr)
				continue
			}
			if !info.Mode().IsRegular() || info.Size() <= 0 {
				t.Errorf("%s is not a non-empty regular file", key)
				continue
			}

			checksum, size, digestErr := digestFile(filePath)
			if digestErr != nil {
				t.Errorf("digest %s: %v", key, digestErr)
				continue
			}
			if checksum != artifact.Checksum {
				t.Errorf("%s checksum does not match database", key)
			}
			if size != artifact.SizeBytes {
				t.Errorf("%s size does not match database", key)
			}
		}

		if len(firstRun) != len(expected) {
			t.Fatalf("verified %d artifact types, want %d", len(firstRun), len(expected))
		}

		if processErr := executor.Process(executorCtx, executorJobID.String(), "cad.export", string(payload)); processErr != nil {
			t.Fatalf("repeat FreeCAD export: %v", processErr)
		}

		repeated, repeatErr := repository.LoadArtifactsByJob(executorCtx, executorJobID.String())
		if repeatErr != nil {
			t.Fatalf("load repeated artifacts: %v", repeatErr)
		}
		if len(repeated) != len(firstRun) {
			t.Fatalf("repeat returned %d artifacts, want %d", len(repeated), len(firstRun))
		}

		for _, artifact := range repeated {
			key := artifact.Dimension + "/" + artifact.Format
			original, exists := firstRun[key]
			if !exists {
				t.Errorf("repeat returned unexpected artifact %s", key)
				continue
			}
			if artifact.ID != original.ID ||
				artifact.StorageKey != original.StorageKey ||
				artifact.Checksum != original.Checksum ||
				artifact.SizeBytes != original.SizeBytes {
				t.Errorf("repeat changed persisted metadata for %s", key)
			}
		}
	})
}
