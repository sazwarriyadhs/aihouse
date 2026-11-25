package cad

import _ "embed"

//go:embed freecad_export.py
var embeddedExporter []byte

// EmbeddedExporter returns an isolated copy of the bundled FreeCAD script.
func EmbeddedExporter() []byte {
	return append([]byte(nil), embeddedExporter...)
}
