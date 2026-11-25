package cad

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type DesignVersion struct {
	ProjectID      string
	DesignID       string
	VersionID      string
	VersionNumber  int
	ModelChecksum  string
	CanonicalModel []byte
	CreatedAt      time.Time
}

type ArtifactInput struct {
	JobID      string
	ProjectID  string
	DesignID   string
	VersionID  string
	Dimension  string
	Format     string
	StorageKey string
	Checksum   string
	SizeBytes  int64
}

type Artifact struct {
	ID         string
	JobID      string
	ProjectID  string
	DesignID   string
	VersionID  string
	Dimension  string
	Format     string
	StorageKey string
	Checksum   string
	SizeBytes  int64
	CreatedAt  time.Time
}

func validateUUID(field, value string) error {
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("%s must be a valid UUID: %w", field, err)
	}
	return nil
}

func (r *Repository) LoadVersion(
	ctx context.Context,
	projectID string,
	designID string,
	versionID string,
) (DesignVersion, error) {
	for field, value := range map[string]string{
		"project_id": projectID,
		"design_id":  designID,
		"version_id": versionID,
	} {
		if err := validateUUID(field, value); err != nil {
			return DesignVersion{}, err
		}
	}

	var version DesignVersion

	err := r.pool.QueryRow(ctx, `
        SELECT
            p.id::text,
            d.id::text,
            v.id::text,
            v.version_number,
            COALESCE(v.model_checksum, ''),
            v.canonical_model,
            v.created_at
        FROM core.design_versions v
        JOIN core.designs d ON d.id = v.design_id
        JOIN core.projects p ON p.id = d.project_id
        WHERE p.id = $1::uuid
          AND d.id = $2::uuid
          AND v.id = $3::uuid
    `, projectID, designID, versionID).Scan(
		&version.ProjectID,
		&version.DesignID,
		&version.VersionID,
		&version.VersionNumber,
		&version.ModelChecksum,
		&version.CanonicalModel,
		&version.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DesignVersion{}, fmt.Errorf(
				"design version not found for supplied project/design/version",
			)
		}
		return DesignVersion{}, fmt.Errorf("load design version: %w", err)
	}

	return version, nil
}

func validateArtifactInput(input ArtifactInput) error {
	for field, value := range map[string]string{
		"job_id":     input.JobID,
		"project_id": input.ProjectID,
		"design_id":  input.DesignID,
		"version_id": input.VersionID,
	} {
		if err := validateUUID(field, value); err != nil {
			return err
		}
	}

	if input.Dimension != "2d" && input.Dimension != "3d" {
		return fmt.Errorf("unsupported CAD dimension: %s", input.Dimension)
	}

	switch {
	case input.Dimension == "2d" &&
		(input.Format == "dxf" || input.Format == "fcstd"):
	case input.Dimension == "3d" &&
		(input.Format == "fcstd" || input.Format == "step" || input.Format == "stl"):
	default:
		return fmt.Errorf(
			"format %q is not supported for dimension %q",
			input.Format,
			input.Dimension,
		)
	}

	if input.StorageKey == "" ||
		strings.Contains(input.StorageKey, "\\") ||
		strings.HasPrefix(input.StorageKey, "/") ||
		path.Clean(input.StorageKey) != input.StorageKey ||
		input.StorageKey == "." ||
		input.StorageKey == ".." ||
		strings.HasPrefix(input.StorageKey, "../") {
		return fmt.Errorf("invalid relative storage key")
	}

	checksum, err := hex.DecodeString(input.Checksum)
	if err != nil || len(checksum) != 32 ||
		strings.ToLower(input.Checksum) != input.Checksum {
		return fmt.Errorf("checksum must be a lowercase SHA256 hex string")
	}

	if input.SizeBytes <= 0 {
		return fmt.Errorf("artifact size must be positive")
	}

	return nil
}

func scanArtifact(row pgx.Row) (Artifact, error) {
	var artifact Artifact
	err := row.Scan(
		&artifact.ID,
		&artifact.JobID,
		&artifact.ProjectID,
		&artifact.DesignID,
		&artifact.VersionID,
		&artifact.Dimension,
		&artifact.Format,
		&artifact.StorageKey,
		&artifact.Checksum,
		&artifact.SizeBytes,
		&artifact.CreatedAt,
	)
	return artifact, err
}

const artifactColumns = `
    id::text,
    job_id::text,
    project_id::text,
    design_id::text,
    version_id::text,
    dimension,
    format,
    storage_key,
    checksum_sha256,
    size_bytes,
    created_at
`

// LoadArtifactsByJob returns the metadata already committed for a job.
func (r *Repository) LoadArtifactsByJob(
	ctx context.Context,
	jobID string,
) ([]Artifact, error) {
	if err := validateUUID("job_id", jobID); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
        SELECT `+artifactColumns+`
        FROM core.cad_artifacts
        WHERE job_id = $1::uuid
        ORDER BY dimension, format
    `, jobID)
	if err != nil {
		return nil, fmt.Errorf("load CAD artifacts for job: %w", err)
	}
	defer rows.Close()

	artifacts := make([]Artifact, 0)
	for rows.Next() {
		artifact, scanErr := scanArtifact(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan CAD artifact: %w", scanErr)
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate CAD artifacts: %w", err)
	}

	return artifacts, nil
}

// CreateArtifact preserves the original single-artifact API.
func (r *Repository) CreateArtifact(
	ctx context.Context,
	input ArtifactInput,
) (Artifact, error) {
	artifacts, err := r.CreateArtifacts(ctx, []ArtifactInput{input})
	if err != nil {
		return Artifact{}, err
	}
	return artifacts[0], nil
}

// CreateArtifacts inserts all outputs in one PostgreSQL transaction.
// Existing rows are accepted only when every identifying field matches.
func (r *Repository) CreateArtifacts(
	ctx context.Context,
	inputs []ArtifactInput,
) ([]Artifact, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("at least one CAD artifact is required")
	}

	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if err := validateArtifactInput(input); err != nil {
			return nil, err
		}

		key := input.JobID + "/" + input.Dimension + "/" + input.Format
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate artifact in batch: %s", key)
		}
		seen[key] = struct{}{}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin CAD artifact batch transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	results := make([]Artifact, 0, len(inputs))

	for _, input := range inputs {
		artifact, insertErr := scanArtifact(tx.QueryRow(ctx, `
            INSERT INTO core.cad_artifacts (
                job_id,
                project_id,
                design_id,
                version_id,
                dimension,
                format,
                storage_key,
                checksum_sha256,
                size_bytes
            )
            VALUES (
                $1::uuid, $2::uuid, $3::uuid, $4::uuid,
                $5, $6, $7, $8, $9
            )
            ON CONFLICT (job_id, dimension, format) DO NOTHING
            RETURNING `+artifactColumns+`
        `,
			input.JobID,
			input.ProjectID,
			input.DesignID,
			input.VersionID,
			input.Dimension,
			input.Format,
			input.StorageKey,
			input.Checksum,
			input.SizeBytes,
		))

		if errors.Is(insertErr, pgx.ErrNoRows) {
			artifact, insertErr = scanArtifact(tx.QueryRow(ctx, `
                SELECT `+artifactColumns+`
                FROM core.cad_artifacts
                WHERE job_id = $1::uuid
                  AND dimension = $2
                  AND format = $3
            `, input.JobID, input.Dimension, input.Format))

			if insertErr != nil {
				return nil, fmt.Errorf("read existing CAD artifact: %w", insertErr)
			}

			if artifact.ProjectID != input.ProjectID ||
				artifact.DesignID != input.DesignID ||
				artifact.VersionID != input.VersionID ||
				artifact.StorageKey != input.StorageKey ||
				artifact.Checksum != input.Checksum ||
				artifact.SizeBytes != input.SizeBytes {
				return nil, fmt.Errorf(
					"artifact idempotency conflict for job %s, dimension %s, format %s",
					input.JobID,
					input.Dimension,
					input.Format,
				)
			}
		} else if insertErr != nil {
			return nil, fmt.Errorf("insert CAD artifact: %w", insertErr)
		}

		results = append(results, artifact)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit CAD artifact batch: %w", err)
	}

	return results, nil
}
