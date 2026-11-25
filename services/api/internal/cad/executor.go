package cad

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type ExportRequest struct {
	ProjectID string `json:"project_id"`
	DesignID  string `json:"design_id"`
	VersionID string `json:"version_id"`
	Target    string `json:"target"`
}

type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

func invalid(format string, args ...any) error {
	return &ValidationError{Err: fmt.Errorf(format, args...)}
}

type Executor struct {
	repo       *Repository
	storageDir string
	pythonPath string
	script     []byte
}

func NewExecutor(repo *Repository, storageDir, pythonPath string, script []byte) (*Executor, error) {
	if repo == nil {
		return nil, fmt.Errorf("CAD repository is required")
	}
	if len(script) == 0 {
		return nil, fmt.Errorf("FreeCAD exporter script is empty")
	}

	if storageDir == "" {
		storageDir = "../../storage/cad"
	}
	absoluteStorage, err := filepath.Abs(storageDir)
	if err != nil {
		return nil, fmt.Errorf("resolve CAD storage directory: %w", err)
	}
	if pythonPath == "" {
		pythonPath = `D:\freecad\bin\python.exe`
	}

	return &Executor{
		repo:       repo,
		storageDir: absoluteStorage,
		pythonPath: pythonPath,
		script:     script,
	}, nil
}

func decodeRequest(payload string) (ExportRequest, error) {
	var req ExportRequest
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		return req, invalid("invalid CAD job payload: %v", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return req, invalid("CAD job payload must contain one JSON object")
	}

	for field, value := range map[string]string{
		"project_id": req.ProjectID,
		"design_id":  req.DesignID,
		"version_id": req.VersionID,
	} {
		if _, err := uuid.Parse(value); err != nil {
			return req, invalid("%s must be a valid UUID", field)
		}
	}
	if req.Target != "2d" && req.Target != "3d" && req.Target != "both" {
		return req, invalid("target must be 2d, 3d, or both")
	}
	return req, nil
}

type modelForExport struct {
	Building struct {
		Floors       int             `json:"floors"`
		FloorPlans   json.RawMessage `json:"floor_plans"`
		Construction struct {
			WallHeight    float64 `json:"wall_height_m"`
			SlabThickness float64 `json:"slab_thickness_m"`
			FloorToFloor  float64 `json:"floor_to_floor_height_m"`
		} `json:"construction"`
		Footprint []struct {
			X float64 `json:"x_m"`
			Y float64 `json:"y_m"`
		} `json:"footprint"`
	} `json:"building"`
}

type cadPoint struct {
	X float64 `json:"x_m"`
	Y float64 `json:"y_m"`
}

type cadWall struct {
	ID        string  `json:"id"`
	X1        float64 `json:"x1_m"`
	Y1        float64 `json:"y1_m"`
	X2        float64 `json:"x2_m"`
	Y2        float64 `json:"y2_m"`
	Thickness float64 `json:"thickness_m"`
}

type cadOpening struct {
	ID     string  `json:"id"`
	WallID string  `json:"wall_id"`
	Offset float64 `json:"offset_m"`
	Width  float64 `json:"width_m"`
	Height float64 `json:"height_m"`
	Sill   float64 `json:"sill_height_m"`
}

type cadFloorPlan struct {
	FloorNumber int               `json:"floor_number"`
	Rooms       []json.RawMessage `json:"rooms"`
	Walls       []cadWall         `json:"walls"`
	Doors       []cadOpening      `json:"doors"`
	Windows     []cadOpening      `json:"windows"`
}

func cadFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func cadCross(a, b, c cadPoint) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func cadOnSegment(a, b, p cadPoint) bool {
	const eps = 1e-9
	return math.Abs(cadCross(a, b, p)) <= eps &&
		p.X >= math.Min(a.X, b.X)-eps &&
		p.X <= math.Max(a.X, b.X)+eps &&
		p.Y >= math.Min(a.Y, b.Y)-eps &&
		p.Y <= math.Max(a.Y, b.Y)+eps
}

func cadSegmentsIntersect(a, b, c, d cadPoint) bool {
	const eps = 1e-9
	o1 := cadCross(a, b, c)
	o2 := cadCross(a, b, d)
	o3 := cadCross(c, d, a)
	o4 := cadCross(c, d, b)

	if ((o1 > eps && o2 < -eps) || (o1 < -eps && o2 > eps)) &&
		((o3 > eps && o4 < -eps) || (o3 < -eps && o4 > eps)) {
		return true
	}
	return cadOnSegment(a, b, c) || cadOnSegment(a, b, d) ||
		cadOnSegment(c, d, a) || cadOnSegment(c, d, b)
}

func validateCADFootprint(points []struct {
	X float64 `json:"x_m"`
	Y float64 `json:"y_m"`
}) error {
	if len(points) < 3 {
		return invalid("3D export requires building.footprint with at least 3 points")
	}

	polygon := make([]cadPoint, 0, len(points)+1)
	for i, p := range points {
		if !cadFinite(p.X) || !cadFinite(p.Y) || p.X < 0 || p.Y < 0 {
			return invalid("building.footprint point %d has invalid coordinates", i)
		}
		polygon = append(polygon, cadPoint{X: p.X, Y: p.Y})
	}

	if polygon[0] == polygon[len(polygon)-1] {
		polygon = polygon[:len(polygon)-1]
	}
	if len(polygon) < 3 {
		return invalid("building.footprint requires at least 3 distinct points")
	}

	area2 := 0.0
	for i := range polygon {
		j := (i + 1) % len(polygon)
		if polygon[i] == polygon[j] {
			return invalid("building.footprint contains duplicate adjacent points")
		}
		area2 += polygon[i].X*polygon[j].Y - polygon[j].X*polygon[i].Y
	}
	if math.Abs(area2) <= 1e-9 {
		return invalid("building.footprint area must be greater than zero")
	}

	n := len(polygon)
	for i := 0; i < n; i++ {
		a, b := polygon[i], polygon[(i+1)%n]
		for j := i + 1; j < n; j++ {
			if j == i || j == (i+1)%n || (j+1)%n == i {
				continue
			}
			c, d := polygon[j], polygon[(j+1)%n]
			if cadSegmentsIntersect(a, b, c, d) {
				return invalid("building.footprint has intersecting edges")
			}
		}
	}
	return nil
}

func validateCADPlan(plan cadFloorPlan, wallHeight float64, require3D bool) error {
	if plan.FloorNumber < 1 {
		return invalid("floor_number must be at least 1")
	}
	if len(plan.Rooms) == 0 && len(plan.Walls) == 0 {
		return invalid("floor %d has no rooms or walls", plan.FloorNumber)
	}

	walls := make(map[string]cadWall, len(plan.Walls))
	for _, wall := range plan.Walls {
		if strings.TrimSpace(wall.ID) == "" {
			return invalid("floor %d contains a wall without an ID", plan.FloorNumber)
		}
		if _, exists := walls[wall.ID]; exists {
			return invalid("floor %d contains duplicate wall ID %q", plan.FloorNumber, wall.ID)
		}
		values := []float64{wall.X1, wall.Y1, wall.X2, wall.Y2, wall.Thickness}
		for _, value := range values {
			if !cadFinite(value) {
				return invalid("wall %q contains a non-finite dimension", wall.ID)
			}
		}
		if wall.X1 < 0 || wall.Y1 < 0 || wall.X2 < 0 || wall.Y2 < 0 ||
			wall.Thickness <= 0 || math.Hypot(wall.X2-wall.X1, wall.Y2-wall.Y1) <= 1e-9 {
			return invalid("wall %q has invalid coordinates, length or thickness", wall.ID)
		}
		walls[wall.ID] = wall
	}

	openingIDs := make(map[string]bool)
	allOpenings := append(append([]cadOpening{}, plan.Doors...), plan.Windows...)
	for _, opening := range allOpenings {
		if strings.TrimSpace(opening.ID) == "" || openingIDs[opening.ID] {
			return invalid("floor %d contains an empty or duplicate opening ID", plan.FloorNumber)
		}
		openingIDs[opening.ID] = true

		wall, found := walls[opening.WallID]
		if !found {
			return invalid("opening %q references missing wall %q", opening.ID, opening.WallID)
		}
		values := []float64{opening.Offset, opening.Width, opening.Height, opening.Sill}
		for _, value := range values {
			if !cadFinite(value) {
				return invalid("opening %q contains a non-finite dimension", opening.ID)
			}
		}

		wallLength := math.Hypot(wall.X2-wall.X1, wall.Y2-wall.Y1)
		if opening.Offset < 0 || opening.Width <= 0 || opening.Height <= 0 ||
			opening.Sill < 0 || opening.Offset+opening.Width > wallLength+1e-9 {
			return invalid("opening %q does not fit its wall", opening.ID)
		}
		if require3D && opening.Sill+opening.Height > wallHeight+1e-9 {
			return invalid("opening %q exceeds the wall height", opening.ID)
		}
	}
	return nil
}

func validateModel(modelJSON []byte, target string) error {
	var model modelForExport
	if err := json.Unmarshal(modelJSON, &model); err != nil {
		return invalid("canonical model is invalid JSON: %v", err)
	}
	if model.Building.Floors < 1 {
		return invalid("building.floors must be at least 1")
	}

	if len(model.Building.FloorPlans) == 0 ||
		string(model.Building.FloorPlans) == "null" {
		return invalid("building.floor_plans is missing")
	}

	var plans []cadFloorPlan
	if err := json.Unmarshal(model.Building.FloorPlans, &plans); err != nil {
		return invalid("building.floor_plans must be an array of valid floor plans: %v", err)
	}
	if len(plans) == 0 {
		return invalid("building.floor_plans is empty")
	}
	if len(plans) > model.Building.Floors {
		return invalid("floor plan count exceeds building.floors")
	}

	seenFloors := make(map[int]bool)
	for _, plan := range plans {
		if plan.FloorNumber < 1 || plan.FloorNumber > model.Building.Floors {
			return invalid("floor number %d is outside building.floors", plan.FloorNumber)
		}
		if seenFloors[plan.FloorNumber] {
			return invalid("duplicate floor number %d", plan.FloorNumber)
		}
		seenFloors[plan.FloorNumber] = true
	}

	is3D := target == "3d" || target == "both"
	construction := model.Building.Construction
	if is3D {
		if !cadFinite(construction.WallHeight) || construction.WallHeight <= 0 {
			return invalid("3D export requires building.construction.wall_height_m > 0")
		}
		if !cadFinite(construction.SlabThickness) || construction.SlabThickness <= 0 {
			return invalid("3D export requires building.construction.slab_thickness_m > 0")
		}
		if model.Building.Floors > 1 &&
			(!cadFinite(construction.FloorToFloor) || construction.FloorToFloor <= 0) {
			return invalid("multi-storey 3D export requires building.construction.floor_to_floor_height_m > 0")
		}
		if model.Building.Floors > 1 && construction.FloorToFloor < construction.WallHeight {
			return invalid("floor_to_floor_height_m cannot be less than wall_height_m")
		}
		if err := validateCADFootprint(model.Building.Footprint); err != nil {
			return err
		}
	}

	hasWalls := false
	for _, plan := range plans {
		if len(plan.Walls) > 0 {
			hasWalls = true
		}
		if err := validateCADPlan(plan, construction.WallHeight, is3D); err != nil {
			return err
		}
	}
	if is3D && !hasWalls {
		return invalid("3D export requires explicit floor-plan walls")
	}
	return nil
}

func digestFile(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func classifyFreeCADFailure(ctx context.Context, processErr error, output []byte) error {
	if ctx.Err() != nil {
		return fmt.Errorf("FreeCAD export interrupted: %w", ctx.Err())
	}

	const marker = "CAD_VALIDATION_ERROR:"

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if strings.HasPrefix(line, marker) {
			detail := strings.TrimSpace(strings.TrimPrefix(line, marker))
			if detail == "" {
				detail = "invalid CAD geometry"
			}
			return invalid("FreeCAD geometry validation failed: %s", detail)
		}
	}

	return fmt.Errorf(
		"FreeCAD export failed: %w: %s",
		processErr,
		strings.TrimSpace(string(output)),
	)
}
func (e *Executor) Process(ctx context.Context, jobID, jobType, payload string) error {
	if jobType != "cad.export" {
		return invalid("CAD executor received unsupported job type: %s", jobType)
	}
	if _, err := uuid.Parse(jobID); err != nil {
		return invalid("job ID must be a valid UUID")
	}

	req, err := decodeRequest(payload)
	if err != nil {
		return err
	}

	version, err := e.repo.LoadVersion(ctx, req.ProjectID, req.DesignID, req.VersionID)
	if err != nil {
		return fmt.Errorf("load immutable design version: %w", err)
	}
	if err := validateModel(version.CanonicalModel, req.Target); err != nil {
		return err
	}

	type outputFile struct {
		Dimension string
		Format    string
		Filename  string
	}

	expected := make([]outputFile, 0, 4)
	if req.Target == "2d" || req.Target == "both" {
		expected = append(expected,
			outputFile{"2d", "dxf", "2d.dxf"},
			outputFile{"2d", "fcstd", "2d.fcstd"},
		)
	}
	if req.Target == "3d" || req.Target == "both" {
		expected = append(expected,
			outputFile{"3d", "fcstd", "3d.fcstd"},
			outputFile{"3d", "step", "3d.step"},
		)
	}

	expectedByKey := make(map[string]outputFile, len(expected))
	for _, item := range expected {
		expectedByKey[item.Dimension+"/"+item.Format] = item
	}

	// Read previously committed metadata before regenerating anything.
	existing, err := e.repo.LoadArtifactsByJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("load existing CAD artifact metadata: %w", err)
	}

	existingByKey := make(map[string]Artifact, len(existing))
	for _, artifact := range existing {
		key := artifact.Dimension + "/" + artifact.Format
		item, supported := expectedByKey[key]
		if !supported {
			return fmt.Errorf("job contains unexpected existing artifact: %s", key)
		}

		relative := filepath.ToSlash(filepath.Join(
			req.ProjectID, req.DesignID, req.VersionID, jobID, item.Filename,
		))

		if artifact.ProjectID != req.ProjectID ||
			artifact.DesignID != req.DesignID ||
			artifact.VersionID != req.VersionID ||
			artifact.StorageKey != relative {
			return fmt.Errorf("existing artifact metadata does not match this export job")
		}

		absolute := filepath.Join(e.storageDir, filepath.FromSlash(artifact.StorageKey))
		checksum, size, digestErr := digestFile(absolute)
		if digestErr != nil {
			return fmt.Errorf("verify previously published artifact %s: %w", key, digestErr)
		}
		if checksum != artifact.Checksum || size != artifact.SizeBytes {
			return fmt.Errorf("previously published artifact is corrupted: %s", key)
		}

		existingByKey[key] = artifact
	}

	missing := make([]outputFile, 0, len(expected))
	for _, item := range expected {
		key := item.Dimension + "/" + item.Format
		if _, found := existingByKey[key]; !found {
			missing = append(missing, item)
		}
	}

	// A completed retry returns without regenerating non-deterministic CAD files.
	if len(missing) == 0 {
		return nil
	}

	if err := os.MkdirAll(e.storageDir, 0750); err != nil {
		return fmt.Errorf("create CAD storage: %w", err)
	}

	stage, err := os.MkdirTemp(e.storageDir, ".staging-"+jobID+"-")
	if err != nil {
		return fmt.Errorf("create CAD staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	modelPath := filepath.Join(stage, "canonical-model.json")
	if err := os.WriteFile(modelPath, version.CanonicalModel, 0600); err != nil {
		return fmt.Errorf("write CAD input model: %w", err)
	}

	scriptPath := filepath.Join(stage, "freecad_export.py")
	if err := os.WriteFile(scriptPath, e.script, 0600); err != nil {
		return fmt.Errorf("write FreeCAD script: %w", err)
	}

	command := exec.CommandContext(
		ctx, e.pythonPath, scriptPath,
		"--input", modelPath,
		"--output", stage,
		"--target", req.Target,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return classifyFreeCADFailure(ctx, err, output)
	}

	inputs := make([]ArtifactInput, 0, len(missing))
	createdPaths := make([]string, 0, len(missing))

	for _, item := range missing {
		source := filepath.Join(stage, item.Filename)
		info, statErr := os.Stat(source)
		if statErr != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
			return fmt.Errorf("FreeCAD did not produce a valid %s artifact", item.Filename)
		}

		checksum, size, digestErr := digestFile(source)
		if digestErr != nil {
			return fmt.Errorf("checksum generated artifact %s: %w", item.Filename, digestErr)
		}

		relative := filepath.ToSlash(filepath.Join(
			req.ProjectID, req.DesignID, req.VersionID, jobID, item.Filename,
		))
		destination := filepath.Join(e.storageDir, filepath.FromSlash(relative))

		if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
			return fmt.Errorf("create artifact directory: %w", err)
		}

		// Reuse a matching orphan from an interrupted earlier attempt.
		// Never overwrite an existing destination without verifying its content.
		if existingInfo, lstatErr := os.Lstat(destination); lstatErr == nil {
			if !existingInfo.Mode().IsRegular() {
				return fmt.Errorf("orphan artifact destination is not a regular file: %s", relative)
			}

			existingChecksum, existingSize, digestErr := digestFile(destination)
			if digestErr != nil {
				return fmt.Errorf("verify orphan artifact %s: %w", relative, digestErr)
			}
			if existingChecksum != checksum || existingSize != size {
				return fmt.Errorf("orphan artifact conflicts with regenerated output: %s", relative)
			}
		} else if errors.Is(lstatErr, os.ErrNotExist) {
			if err := os.Rename(source, destination); err != nil {
				return fmt.Errorf("publish artifact %s: %w", relative, err)
			}
			createdPaths = append(createdPaths, destination)
		} else {
			return fmt.Errorf("inspect artifact destination %s: %w", relative, lstatErr)
		}
		inputs = append(inputs, ArtifactInput{
			JobID:      jobID,
			ProjectID:  req.ProjectID,
			DesignID:   req.DesignID,
			VersionID:  req.VersionID,
			Dimension:  item.Dimension,
			Format:     item.Format,
			StorageKey: relative,
			Checksum:   checksum,
			SizeBytes:  size,
		})
	}

	if _, err := e.repo.CreateArtifacts(ctx, inputs); err != nil {
		// Resolve an ambiguous commit result before removing any files.
		// If metadata exists, preserve its corresponding files.
		persisted, lookupErr := e.repo.LoadArtifactsByJob(ctx, jobID)
		if lookupErr == nil {
			persistedKeys := make(map[string]bool, len(persisted))
			for _, artifact := range persisted {
				persistedKeys[artifact.StorageKey] = true
			}
			for _, filePath := range createdPaths {
				relative, relErr := filepath.Rel(e.storageDir, filePath)
				if relErr == nil && !persistedKeys[filepath.ToSlash(relative)] {
					_ = os.Remove(filePath)
				}
			}
		}
		return fmt.Errorf("persist CAD artifact batch: %w", err)
	}

	return nil
}
