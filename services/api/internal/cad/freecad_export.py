import argparse
import json
import math
import os
import sys

import FreeCAD as App
import Part
import importDXF

class CADValidationError(Exception):
    pass

def require(condition, message):
    if not condition:
        raise CADValidationError(message)

def vector(x, y, z=0):
    return App.Vector(float(x) * 1000.0, float(y) * 1000.0, float(z) * 1000.0)

def make_wire(points):
    vertices = [vector(p["x_m"], p["y_m"]) for p in points]
    require(len(vertices) >= 3, "footprint requires at least three points")
    if vertices[0].distanceToPoint(vertices[-1]) > 0.001:
        vertices.append(vertices[0])
    return Part.makePolygon(vertices)

def add_feature(doc, name, shape):
    obj = doc.addObject("Part::Feature", name)
    obj.Shape = shape
    return obj

def room_outline(room):
    x = float(room["x_m"])
    y = float(room["y_m"])
    w = float(room["width_m"])
    d = float(room["depth_m"])
    require(w > 0 and d > 0, "room dimensions must be positive")
    pts = [vector(x,y), vector(x+w,y), vector(x+w,y+d), vector(x,y+d), vector(x,y)]
    return Part.makePolygon(pts)

def build_2d(doc, model):
    objects = []
    plans = model["building"]["floor_plans"]
    for plan in plans:
        floor = int(plan["floor_number"])
        for room in plan.get("rooms", []):
            shape = room_outline(room)
            objects.append(add_feature(doc, "RoomBoundary_F%d_%s" % (floor, room["id"]), shape))
        for wall in plan.get("walls", []):
            a = vector(wall["x1_m"], wall["y1_m"])
            b = vector(wall["x2_m"], wall["y2_m"])
            require(a.distanceToPoint(b) > 0, "wall length must be positive")
            shape = Part.makePolygon([a,b])
            objects.append(add_feature(doc, "WallCenterline_F%d_%s" % (floor, wall["id"]), shape))
    require(objects, "no 2D geometry found")
    return objects

def wall_solid(wall, height, floor_z, openings):
    x1 = float(wall["x1_m"])
    y1 = float(wall["y1_m"])
    x2 = float(wall["x2_m"])
    y2 = float(wall["y2_m"])
    thickness = float(wall["thickness_m"])

    dx = x2 - x1
    dy = y2 - y1
    length = math.hypot(dx, dy)

    require(length > 0 and thickness > 0, "invalid wall geometry")

    angle = math.degrees(math.atan2(dy, dx))

    shape = Part.makeBox(
        length * 1000.0,
        thickness * 1000.0,
        height * 1000.0,
        App.Vector(0, -thickness * 500.0, floor_z * 1000.0)
    )

    # Cut every door/window opening in the wall's local coordinate system.
    # The cutter extends 20 mm beyond both wall faces to avoid coplanar cuts.
    for opening in openings:
        offset = float(opening["offset_m"])
        width = float(opening["width_m"])
        opening_height = float(opening["height_m"])
        sill = float(opening.get("sill_height_m", 0.0))

        require(offset >= 0 and width > 0 and opening_height > 0 and sill >= 0,
                "invalid opening dimensions")
        require(offset + width <= length + 1e-9,
                "opening exceeds wall length")
        require(sill + opening_height <= height + 1e-9,
                "opening exceeds wall height")

        cutter = Part.makeBox(
            width * 1000.0,
            (thickness * 1000.0) + 40.0,
            opening_height * 1000.0,
            App.Vector(
                offset * 1000.0,
                -(thickness * 500.0) - 20.0,
                (floor_z + sill) * 1000.0
            )
        )

        # Both wall and cutter are still in the wall-local coordinate
        # system here. Transforming the cutter before the boolean cut
        # would make it miss the untransformed wall.
        shape = shape.cut(cutter)

        require(not shape.isNull(), "wall became null after opening cut")
        require(shape.isValid(), "wall became invalid after opening cut")
        require(shape.Volume > 0, "wall has no remaining volume")

    shape.rotate(
        App.Vector(0, 0, 0),
        App.Vector(0, 0, 1),
        angle
    )
    shape.translate(vector(x1, y1))

    require(not shape.isNull() and shape.isValid(),
            "generated wall solid is invalid")
    require(shape.Volume > 0, "generated wall solid has zero volume")

    return shape
def build_3d(doc, model):
    building = model["building"]
    construction = building["construction"]

    wall_height = float(construction["wall_height_m"])
    slab_thickness = float(construction["slab_thickness_m"])
    floors = int(building["floors"])
    floor_to_floor = float(construction.get("floor_to_floor_height_m", 0))

    require(wall_height > 0 and slab_thickness > 0,
            "invalid construction dimensions")

    if floors > 1:
        require(floor_to_floor >= wall_height,
                "floor_to_floor_height_m cannot be less than wall_height_m")

    footprint = building["footprint"]
    wire = make_wire(footprint)
    face = Part.Face(Part.Wire(wire.Edges))

    require(not face.isNull() and face.isValid(),
            "building footprint cannot form a valid face")

    slab = face.extrude(App.Vector(0, 0, -slab_thickness * 1000.0))
    require(not slab.isNull() and slab.isValid() and slab.Volume > 0,
            "building slab generation failed")

    objects = [add_feature(doc, "BuildingSlab", slab)]

    for plan in building["floor_plans"]:
        floor = int(plan["floor_number"])
        floor_z = (floor - 1) * floor_to_floor

        walls = plan.get("walls", [])
        wall_ids = set()

        # Every wall ID must be unique within this floor plan.
        for wall in walls:
            wall_id = wall["id"]
            require(wall_id not in wall_ids,
                    "duplicate wall ID: %s" % wall_id)
            wall_ids.add(wall_id)

        openings_by_wall = {}

        # Reject orphan openings instead of silently ignoring them.
        for opening in plan.get("doors", []) + plan.get("windows", []):
            wall_id = opening.get("wall_id")
            require(
                isinstance(wall_id, str) and wall_id in wall_ids,
                "opening references unknown wall: %s" % wall_id
            )
            openings_by_wall.setdefault(wall_id, []).append(opening)

        for wall in walls:
            openings = openings_by_wall.get(wall["id"], [])
            solid = wall_solid(
                wall,
                wall_height,
                floor_z,
                openings
            )

            objects.append(add_feature(
                doc,
                "Wall_F%d_%s" % (floor, wall["id"]),
                solid
            ))

    require(len(objects) > 1, "no 3D wall solids generated")
    return objects
def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--target", choices=["2d", "3d", "both"], required=True)
    args = parser.parse_args()

    with open(args.input, "r", encoding="utf-8") as handle:
        model = json.load(handle)

    doc = App.newDocument("AIHouseGenerator")
    try:
        if args.target in ("2d", "both"):
            objects_2d = build_2d(doc, model)
            doc.recompute()
            doc.saveAs(os.path.join(args.output, "2d.fcstd"))
            importDXF.export(objects_2d, os.path.join(args.output, "2d.dxf"))
            for obj in objects_2d:
                doc.removeObject(obj.Name)
            doc.recompute()

        if args.target in ("3d", "both"):
            objects_3d = build_3d(doc, model)
            doc.recompute()
            doc.saveAs(os.path.join(args.output, "3d.fcstd"))
            Part.export(objects_3d, os.path.join(args.output, "3d.step"))

    finally:
        App.closeDocument(doc.Name)

if __name__ == "__main__":
    try:
        main()
    except CADValidationError as exc:
        print("CAD_VALIDATION_ERROR: " + str(exc), file=sys.stderr)
        sys.exit(2)
    except Exception as exc:
        print("CAD_EXPORT_ERROR: " + str(exc), file=sys.stderr)
        raise
