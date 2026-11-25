package cad

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func testCADModel() []byte {
	model := map[string]any{
		"building": map[string]any{
			"floors": 1,
			"construction": map[string]any{
				"wall_height_m":    3.0,
				"slab_thickness_m": 0.15,
			},
			"footprint": []any{
				map[string]any{"x_m": 0.0, "y_m": 0.0},
				map[string]any{"x_m": 8.0, "y_m": 0.0},
				map[string]any{"x_m": 8.0, "y_m": 10.0},
				map[string]any{"x_m": 0.0, "y_m": 10.0},
			},
			"floor_plans": []any{
				map[string]any{
					"floor_number": 1,
					"rooms":        []any{map[string]any{"id": "room-1", "x_m": 1.0, "y_m": 1.0, "width_m": 3.0, "depth_m": 2.0}},
					"walls": []any{
						map[string]any{
							"id":   "wall-1",
							"x1_m": 1.0, "y1_m": 1.0,
							"x2_m": 5.0, "y2_m": 1.0,
							"thickness_m": 0.15,
						},
					},
					"doors": []any{
						map[string]any{
							"id": "door-1", "wall_id": "wall-1",
							"offset_m": 0.5, "width_m": 0.9,
							"height_m": 2.1,
						},
					},
					"windows": []any{},
				},
			},
		},
	}
	data, _ := json.Marshal(model)
	return data
}

func TestValidateModelAcceptsValid2DAnd3D(t *testing.T) {
	data := testCADModel()
	for _, target := range []string{"2d", "3d", "both"} {
		if err := validateModel(data, target); err != nil {
			t.Fatalf("target %s should be valid: %v", target, err)
		}
	}
}

func TestValidateModelRejectsSelfIntersectingFootprint(t *testing.T) {
	data := testCADModel()
	data = []byte(strings.Replace(
		string(data),
		`"x_m":8,"y_m":0`,
		`"x_m":8,"y_m":10`,
		1,
	))
	if err := validateModel(data, "3d"); err == nil {
		t.Fatal("expected self-intersecting footprint to be rejected")
	}
}

func TestValidateModelRejectsOpeningOutsideWall(t *testing.T) {
	data := testCADModel()
	data = []byte(strings.Replace(
		string(data),
		`"width_m":0.9`,
		`"width_m":5.0`,
		1,
	))
	if err := validateModel(data, "3d"); err == nil {
		t.Fatal("expected opening outside wall to be rejected")
	}
}

func TestValidateModelRejectsOpeningAboveWallHeight(t *testing.T) {
	data := testCADModel()
	data = []byte(strings.Replace(
		string(data),
		`"height_m":2.1`,
		`"height_m":3.2`,
		1,
	))
	if err := validateModel(data, "3d"); err == nil {
		t.Fatal("expected opening above wall height to be rejected")
	}
}

func TestValidateModelRejectsMissingOpeningWall(t *testing.T) {
	data := testCADModel()
	data = []byte(strings.Replace(
		string(data),
		`"wall_id":"wall-1"`,
		`"wall_id":"missing-wall"`,
		1,
	))
	if err := validateModel(data, "2d"); err == nil {
		t.Fatal("expected opening referencing missing wall to be rejected")
	}
}

func TestValidateModelRequiresExplicit3DConstruction(t *testing.T) {
	data := testCADModel()
	data = []byte(strings.Replace(
		string(data),
		`"wall_height_m":3`,
		`"wall_height_m":0`,
		1,
	))
	if err := validateModel(data, "3d"); err == nil {
		t.Fatal("expected missing wall height to be rejected")
	}
}

// Validation errors from FreeCAD must become permanent CAD errors.
func TestClassifyFreeCADValidationFailure(t *testing.T) {
	err := classifyFreeCADFailure(
		context.Background(),
		errors.New("exit status 2"),
		[]byte("CAD_VALIDATION_ERROR: opening exceeds wall length"),
	)

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if !strings.Contains(err.Error(), "opening exceeds wall length") {
		t.Fatalf("unexpected validation message: %v", err)
	}
}

func TestClassifyFreeCADRuntimeFailure(t *testing.T) {
	processErr := errors.New("exit status 1")
	err := classifyFreeCADFailure(
		context.Background(),
		processErr,
		[]byte("CAD_EXPORT_ERROR: FreeCAD internal failure"),
	)

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		t.Fatalf("runtime failure must not be permanent: %v", err)
	}

	if !errors.Is(err, processErr) {
		t.Fatalf("original process error was not preserved: %v", err)
	}
}

func TestClassifyFreeCADCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := classifyFreeCADFailure(
		ctx,
		errors.New("signal: killed"),
		[]byte("CAD_VALIDATION_ERROR: arbitrary output"),
	)

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		t.Fatalf("cancelled process must not become permanent: %v", err)
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got: %v", err)
	}
}
