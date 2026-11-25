"use client";

import { useEffect, useId, useMemo, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import {
  AlertCircle,
  CheckCircle2,
  LoaderCircle,
  Plus,
  Save,
  Trash2,
} from "lucide-react";
import {
  getFloorPlans,
  saveFloorPlans,
  type FloorPlan,
  type FloorPlanRoom,
  type FloorPlanWall,
} from "@/lib/api/projects";

type Props = {
  projectId: string;
  landWidth: number | null;
  landDepth: number | null;
  floors: number | null;
};

const SNAP_SIZE = 0.25;
const DEFAULT_WALL_THICKNESS = 0.15;

const ROOM_TYPES = [
  "Living Room",
  "Bedroom",
  "Master Bedroom",
  "Bathroom",
  "Kitchen",
  "Dining Room",
  "Garage",
  "Office",
  "Storage",
  "Other",
];

function emptyPlan(floorNumber: number): FloorPlan {
  return {
    floor_number: floorNumber,
    name: `Floor ${floorNumber}`,
    rooms: [],
    walls: [],
    doors: [],
    windows: [],
  };
}

function makeRoom(
  index: number,
  landWidth: number,
  landDepth: number,
): FloorPlanRoom {
  const width = Math.min(3, landWidth);
  const depth = Math.min(3, landDepth);

  return {
    id: `room-${Date.now()}-${index}`,
    name: `Room ${index + 1}`,
    type: "Bedroom",
    x_m: 0,
    y_m: 0,
    width_m: width,
    depth_m: depth,
  };
}

export default function FloorPlanEditor({
  projectId,
  landWidth,
  landDepth,
  floors,
}: Props) {
  const wallIdPrefix = useId();
  const wallIdCounter = useRef(0);

  const [plans, setPlans] = useState<FloorPlan[]>([]);
  const [activeFloor, setActiveFloor] = useState(1);
  const [selectedRoomId, setSelectedRoomId] = useState<string | null>(null);
  const [selectedWallId, setSelectedWallId] = useState<string | null>(null);
  const [wallDrawingMode, setWallDrawingMode] = useState(false);
  const [draftWall, setDraftWall] = useState<{
    x1: number; y1: number; x2: number; y2: number;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [version, setVersion] = useState<number | null>(null);

  const dragRef = useRef<{
    roomId: string;
    offsetX: number;
    offsetY: number;
  } | null>(null);

  const wallDrawRef = useRef<{ x1: number; y1: number } | null>(null);

  const validDimensions =
    landWidth !== null &&
    landDepth !== null &&
    Number.isFinite(landWidth) &&
    Number.isFinite(landDepth) &&
    landWidth > 0 &&
    landDepth > 0;

  const floorCount =
    floors !== null && Number.isInteger(floors) && floors > 0
      ? Math.min(floors, 10)
      : 1;

  useEffect(() => {
    let active = true;

    getFloorPlans(projectId).then((result) => {
      if (!active) return;

      if (result.error || !result.data) {
        setLoadError(result.error || "Unable to load saved floor plans.");
        setLoading(false);
        return;
      }

      setPlans(result.data.floor_plans);
      setVersion(result.data.version_number);
      setLoading(false);
    });

    return () => {
      active = false;
    };
  }, [projectId]);

  const currentPlan = useMemo(
    () => plans.find((plan) => plan.floor_number === activeFloor)
      ?? emptyPlan(activeFloor),
    [plans, activeFloor],
  );

  const selectedRoom = currentPlan.rooms.find(
    (room) => room.id === selectedRoomId,
  ) ?? null;

  const selectedWall = currentPlan.walls.find(
    (wall) => wall.id === selectedWallId,
  ) ?? null;

  function updateCurrentPlan(nextPlan: FloorPlan) {
    setPlans((previous) => {
      const exists = previous.some(
        (plan) => plan.floor_number === activeFloor,
      );

      if (exists) {
        return previous.map((plan) =>
          plan.floor_number === activeFloor ? nextPlan : plan,
        );
      }

      return [...previous, nextPlan].sort(
        (a, b) => a.floor_number - b.floor_number,
      );
    });

    setSaveMessage(null);
    setSaveError(null);
  }

  function updateRoom(
    roomId: string,
    changes: Partial<FloorPlanRoom>,
  ) {
    const nextRooms = currentPlan.rooms.map((room) => {
      if (room.id !== roomId) return room;

      const next = { ...room, ...changes };

      if (validDimensions) {
        next.width_m = Math.min(
          landWidth!,
          Math.max(0.1, next.width_m),
        );
        next.depth_m = Math.min(
          landDepth!,
          Math.max(0.1, next.depth_m),
        );
        next.x_m = Math.min(
          landWidth! - next.width_m,
          Math.max(0, next.x_m),
        );
        next.y_m = Math.min(
          landDepth! - next.depth_m,
          Math.max(0, next.y_m),
        );
      }

      return next;
    });

    updateCurrentPlan({ ...currentPlan, rooms: nextRooms });
  }

  function getSvgCoordinates(
    svg: SVGSVGElement,
    clientX: number,
    clientY: number,
  ) {
    const matrix = svg.getScreenCTM();
    if (!matrix) return null;

    const point = svg.createSVGPoint();
    point.x = clientX;
    point.y = clientY;

    const transformed = point.matrixTransform(matrix.inverse());

    return {
      x: transformed.x,
      y: transformed.y,
    };
  }

  function handleRoomPointerDown(
    event: ReactPointerEvent<SVGGElement>,
    room: FloorPlanRoom,
  ) {
    if (wallDrawingMode) return;

    const svg = event.currentTarget.ownerSVGElement;
    if (!svg) return;

    const point = getSvgCoordinates(
      svg,
      event.clientX,
      event.clientY,
    );

    if (!point) return;

    event.preventDefault();
    event.stopPropagation();

    setSelectedRoomId(room.id);

    dragRef.current = {
      roomId: room.id,
      offsetX: point.x - room.x_m,
      offsetY: point.y - room.y_m,
    };

    svg.setPointerCapture(event.pointerId);
  }

  function snapCoordinate(value: number, maximum: number) {
    return Math.min(
      maximum,
      Math.max(0, Math.round(value / SNAP_SIZE) * SNAP_SIZE),
    );
  }

  function handleCanvasPointerDown(
    event: ReactPointerEvent<SVGSVGElement>,
  ) {
    if (!wallDrawingMode || !validDimensions) return;

    const point = getSvgCoordinates(
      event.currentTarget,
      event.clientX,
      event.clientY,
    );

    if (!point) return;

    event.preventDefault();

    const x = snapCoordinate(point.x, landWidth!);
    const y = snapCoordinate(point.y, landDepth!);

    wallDrawRef.current = { x1: x, y1: y };
    setDraftWall({ x1: x, y1: y, x2: x, y2: y });

    setSelectedRoomId(null);
    setSelectedWallId(null);

    event.currentTarget.setPointerCapture(event.pointerId);
  }

  function handleCanvasPointerMove(
    event: ReactPointerEvent<SVGSVGElement>,
  ) {
    const wallStart = wallDrawRef.current;

    if (wallStart) {
      const point = getSvgCoordinates(
        event.currentTarget,
        event.clientX,
        event.clientY,
      );

      if (!point) return;

      setDraftWall({
        ...wallStart,
        x2: snapCoordinate(point.x, landWidth!),
        y2: snapCoordinate(point.y, landDepth!),
      });

      return;
    }

    const drag = dragRef.current;
    if (!drag) return;

    const point = getSvgCoordinates(
      event.currentTarget,
      event.clientX,
      event.clientY,
    );

    if (!point) return;

    const room = currentPlan.rooms.find(
      (item) => item.id === drag.roomId,
    );

    if (!room) return;

    const snappedX =
      Math.round((point.x - drag.offsetX) / SNAP_SIZE) * SNAP_SIZE;

    const snappedY =
      Math.round((point.y - drag.offsetY) / SNAP_SIZE) * SNAP_SIZE;

    updateRoom(drag.roomId, {
      x_m: snappedX,
      y_m: snappedY,
    });
  }

  function handleCanvasPointerUp(
    event: ReactPointerEvent<SVGSVGElement>,
  ) {
    dragRef.current = null;

    const start = wallDrawRef.current;
    wallDrawRef.current = null;

    if (event.type === "pointercancel") {
      setDraftWall(null);
    } else if (start && draftWall) {
      const x2 = snapCoordinate(draftWall.x2, landWidth!);
      const y2 = snapCoordinate(draftWall.y2, landDepth!);
      const length = Math.hypot(x2 - start.x1, y2 - start.y1);

      if (length >= SNAP_SIZE) {
        const wall: FloorPlanWall = {
          id: `${wallIdPrefix}-wall-${wallIdCounter.current++}`,
          x1_m: start.x1,
          y1_m: start.y1,
          x2_m: x2,
          y2_m: y2,
          thickness_m: DEFAULT_WALL_THICKNESS,
        };

        updateCurrentPlan({
          ...currentPlan,
          walls: [...currentPlan.walls, wall],
        });

        setSelectedRoomId(null);
        setSelectedWallId(wall.id);
      }

      setDraftWall(null);
    }

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
  }

  function selectWall(
    event: ReactPointerEvent<SVGLineElement>,
    wall: FloorPlanWall,
  ) {
    if (wallDrawingMode) return;

    event.preventDefault();
    event.stopPropagation();

    setSelectedWallId(wall.id);
    setSelectedRoomId(null);
  }

  function updateWall(
    wallId: string,
    changes: Partial<FloorPlanWall>,
  ) {
    updateCurrentPlan({
      ...currentPlan,
      walls: currentPlan.walls.map((wall) =>
        wall.id === wallId ? { ...wall, ...changes } : wall,
      ),
    });
  }

  function deleteWall() {
    if (!selectedWallId) return;

    updateCurrentPlan({
      ...currentPlan,
      walls: currentPlan.walls.filter(
        (wall) => wall.id !== selectedWallId,
      ),
      doors: currentPlan.doors.filter(
        (opening) => opening.wall_id !== selectedWallId,
      ),
      windows: currentPlan.windows.filter(
        (opening) => opening.wall_id !== selectedWallId,
      ),
    });

    setSelectedWallId(null);
  }

  function addRoom() {
    if (!validDimensions) return;

    const room = makeRoom(currentPlan.rooms.length, landWidth!, landDepth!);

    updateCurrentPlan({
      ...currentPlan,
      rooms: [...currentPlan.rooms, room],
    });

    setSelectedRoomId(room.id);
  }

  function deleteRoom() {
    if (!selectedRoomId) return;

    updateCurrentPlan({
      ...currentPlan,
      rooms: currentPlan.rooms.filter(
        (room) => room.id !== selectedRoomId,
      ),
    });

    setSelectedRoomId(null);
  }

  async function handleSave() {
    if (!validDimensions || loading || saving || loadError) return;

    setSaving(true);
    setSaveMessage(null);
    setSaveError(null);

    try {
      const result = await saveFloorPlans(
        projectId,
        [...plans].sort((a, b) => a.floor_number - b.floor_number),
      );

      if (result.error || !result.data) {
        setSaveError(result.error || "Unable to save floor plans.");
        return;
      }

      setPlans(result.data.floor_plans);
      setVersion(result.data.version_number);
      setSaveMessage(
        `Floor plans saved successfully. Design version v${result.data.version_number}.`,
      );
    } catch {
      setSaveError("An unexpected error occurred while saving.");
    } finally {
      setSaving(false);
    }
  }

  function numberField(
    label: string,
    value: number,
    onChange: (value: number) => void,
    step = 0.1,
  ) {
    return (
      <label className="ahg-floor-field">
        <span>{label}</span>
        <input
          type="number"
          min="0"
          step={step}
          value={Number.isFinite(value) ? value : 0}
          onChange={(event) => {
            const next = Number(event.target.value);
            if (Number.isFinite(next)) onChange(next);
          }}
        />
      </label>
    );
  }

  return (
    <section className="ahg-floor-editor">
      <div className="ahg-floor-editor-heading">
        <div>
          <span className="project-wow-eyebrow">INTERACTIVE DESIGN TOOL</span>
          <h2>2D Floor Plan Editor</h2>
          <p>
            Arrange rooms and define their dimensions using your actual land
            measurements.
          </p>
        </div>

        <div className="ahg-floor-version">
          {version !== null ? `Design v${version}` : "Loading version"}
        </div>
      </div>

      {loading && (
        <div className="ahg-floor-notice">
          <LoaderCircle className="ahg-spin" size={18} />
          Loading saved floor plans...
        </div>
      )}

      {loadError && (
        <div className="ahg-floor-notice ahg-floor-error">
          <AlertCircle size={18} />
          {loadError}
        </div>
      )}

      {!validDimensions && !loading && (
        <div className="ahg-floor-notice ahg-floor-error">
          <AlertCircle size={18} />
          Valid land width and depth are required before editing.
          Update the project information first.
        </div>
      )}

      {!loading && !loadError && validDimensions && (
        <>
          <div className="ahg-floor-toolbar">
            <div className="ahg-floor-tabs">
              {Array.from({ length: floorCount }, (_, index) => index + 1).map(
                (floor) => (
                  <button
                    key={floor}
                    type="button"
                    className={
                      activeFloor === floor
                        ? "ahg-floor-tab active"
                        : "ahg-floor-tab"
                    }
                    onClick={() => {
                      setActiveFloor(floor);
                      setSelectedRoomId(null);
                      setSelectedWallId(null);
                      setWallDrawingMode(false);
                      setDraftWall(null);
                      wallDrawRef.current = null;
                      setSaveMessage(null);
                    }}
                  >
                    Floor {floor}
                  </button>
                ),
              )}
            </div>

            <button
              type="button"
              className="ahg-floor-add-button"
              onClick={addRoom}
            >
              <Plus size={16} />
              Add Room
            </button>

            <button
              type="button"
              className={wallDrawingMode
                ? "ahg-floor-wall-mode active"
                : "ahg-floor-wall-mode"}
              onClick={() => {
                setWallDrawingMode((active) => !active);
                setSelectedRoomId(null);
                setSelectedWallId(null);
                setDraftWall(null);
                wallDrawRef.current = null;
              }}
              aria-pressed={wallDrawingMode}
            >
              {wallDrawingMode ? "Cancel Wall Drawing" : "Draw Wall"}
            </button>
          </div>

          <div className="ahg-floor-workspace">
            <div className="ahg-floor-canvas-panel">
              <div className="ahg-floor-canvas-heading">
                <strong>{currentPlan.name}</strong>
                <span>
                  {landWidth} m × {landDepth} m
                </span>
              </div>

              <div className="ahg-floor-canvas">
                <svg
                  viewBox={`0 0 ${landWidth} ${landDepth}`}
                  role="img"
                  aria-label={`Floor ${activeFloor} room layout`}
                  preserveAspectRatio="xMidYMid meet"
                  onPointerDown={handleCanvasPointerDown}
                  onPointerMove={handleCanvasPointerMove}
                  onPointerUp={handleCanvasPointerUp}
                  onPointerCancel={handleCanvasPointerUp}
                >
                  <defs>
                    <pattern
                      id="ahg-floor-grid"
                      width="1"
                      height="1"
                      patternUnits="userSpaceOnUse"
                    >
                      <path
                        d="M 1 0 L 0 0 0 1"
                        fill="none"
                        stroke="#DCE2EA"
                        strokeWidth="0.015"
                      />
                    </pattern>
                  </defs>

                  <rect
                    x="0"
                    y="0"
                    width={landWidth}
                    height={landDepth}
                    fill="#FAFBFD"
                    stroke="#8994A5"
                    strokeWidth="0.06"
                  />

                  <rect
                    x="0"
                    y="0"
                    width={landWidth}
                    height={landDepth}
                    fill="url(#ahg-floor-grid)"
                  />

                  {currentPlan.walls.map((wall) => (
                    <line
                      key={wall.id}
                      x1={wall.x1_m}
                      y1={wall.y1_m}
                      x2={wall.x2_m}
                      y2={wall.y2_m}
                      stroke={selectedWallId === wall.id ? "#1070FF" : "#344256"}
                      strokeWidth={Math.max(wall.thickness_m, 0.08)}
                      strokeLinecap="square"
                      onPointerDown={(event) => selectWall(event, wall)}
                      style={{ cursor: wallDrawingMode ? "crosshair" : "pointer" }}
                    />
                  ))}

                  {draftWall && wallDrawingMode && (
                    <line
                      x1={draftWall.x1}
                      y1={draftWall.y1}
                      x2={draftWall.x2}
                      y2={draftWall.y2}
                      stroke="#1070FF"
                      strokeWidth={DEFAULT_WALL_THICKNESS}
                      strokeDasharray="0.15 0.1"
                      pointerEvents="none"
                    />
                  )}

                  {currentPlan.rooms.map((room) => (
                    <g
                      key={room.id}
                      onPointerDown={(event) =>
                        handleRoomPointerDown(event, room)
                      }
                      style={{ cursor: "grab", touchAction: "none" }}
                    >
                      <rect
                        x={room.x_m}
                        y={room.y_m}
                        width={room.width_m}
                        height={room.depth_m}
                        fill={
                          selectedRoomId === room.id ? "#DCE9FF" : "#EDF3FC"
                        }
                        stroke={
                          selectedRoomId === room.id ? "#1070FF" : "#52647B"
                        }
                        strokeWidth="0.055"
                      />
                      <text
                        x={room.x_m + room.width_m / 2}
                        y={room.y_m + room.depth_m / 2}
                        textAnchor="middle"
                        dominantBaseline="middle"
                        fontSize={Math.max(
                          0.2,
                          Math.min(room.width_m, room.depth_m) * 0.12,
                        )}
                        fill="#243247"
                        style={{ pointerEvents: "none" }}
                      >
                        {room.name.length > 18
                          ? `${room.name.slice(0, 16)}...`
                          : room.name}
                      </text>
                      <text
                        x={room.x_m + room.width_m / 2}
                        y={room.y_m + room.depth_m / 2 + 0.3}
                        textAnchor="middle"
                        fontSize={0.18}
                        fill="#52647B"
                        style={{ pointerEvents: "none" }}
                      >
                        {room.width_m.toFixed(1)} × {room.depth_m.toFixed(1)} m
                      </text>
                    </g>
                  ))}
                </svg>
              </div>

              <div className="ahg-floor-canvas-footer">
                <span>{currentPlan.rooms.length} rooms · {currentPlan.walls.length} walls</span>
                <span>Snap 0.25 m · Dimensions in meters</span>
              </div>
            </div>

            <aside className="ahg-floor-properties">
              <div className="ahg-floor-properties-heading">
                <div>
                  <span className="project-wow-eyebrow">ROOM PROPERTIES</span>
                  <h3>{selectedRoom ? "Edit Room" : "Room Details"}</h3>
                </div>
              </div>

              {selectedWall ? (
                <>
                  <label className="ahg-floor-field">
                    <span>Wall thickness (m)</span>
                    <input
                      type="number"
                      min="0.05"
                      max="1"
                      step="0.05"
                      value={selectedWall.thickness_m}
                      onChange={(event) =>
                        updateWall(selectedWall.id, {
                          thickness_m: Math.max(0.05, Number(event.target.value) || 0.05),
                        })
                      }
                    />
                  </label>

                  <div className="ahg-floor-wall-coordinates">
                    <span>Start: {selectedWall.x1_m.toFixed(2)}, {selectedWall.y1_m.toFixed(2)} m</span>
                    <span>End: {selectedWall.x2_m.toFixed(2)}, {selectedWall.y2_m.toFixed(2)} m</span>
                    <span>Length: {Math.hypot(
                      selectedWall.x2_m - selectedWall.x1_m,
                      selectedWall.y2_m - selectedWall.y1_m,
                    ).toFixed(2)} m</span>
                  </div>

                  <button
                    type="button"
                    className="ahg-floor-delete-button"
                    onClick={deleteWall}
                  >
                    <Trash2 size={16} />
                    Delete Wall
                  </button>
                </>
              ) : selectedRoom ? (
                <>
                  <label className="ahg-floor-field">
                    <span>Room name</span>
                    <input
                      type="text"
                      maxLength={80}
                      value={selectedRoom.name}
                      onChange={(event) =>
                        updateRoom(selectedRoom.id, {
                          name: event.target.value,
                        })
                      }
                    />
                  </label>

                  <label className="ahg-floor-field">
                    <span>Room type</span>
                    <select
                      value={selectedRoom.type}
                      onChange={(event) =>
                        updateRoom(selectedRoom.id, {
                          type: event.target.value,
                        })
                      }
                    >
                      {ROOM_TYPES.map((type) => (
                        <option key={type} value={type}>
                          {type}
                        </option>
                      ))}
                    </select>
                  </label>

                  <div className="ahg-floor-field-grid">
                    {numberField("Position X (m)", selectedRoom.x_m, (value) =>
                      updateRoom(selectedRoom.id, { x_m: value }),
                    )}
                    {numberField("Position Y (m)", selectedRoom.y_m, (value) =>
                      updateRoom(selectedRoom.id, { y_m: value }),
                    )}
                    {numberField("Width (m)", selectedRoom.width_m, (value) =>
                      updateRoom(selectedRoom.id, { width_m: value }),
                    )}
                    {numberField("Depth (m)", selectedRoom.depth_m, (value) =>
                      updateRoom(selectedRoom.id, { depth_m: value }),
                    )}
                  </div>

                  <button
                    type="button"
                    className="ahg-floor-delete-button"
                    onClick={deleteRoom}
                  >
                    <Trash2 size={16} />
                    Delete Room
                  </button>
                </>
              ) : (
                <div className="ahg-floor-empty-selection">
                  <Plus size={22} />
                  <p>Select a room or wall on the floor plan, add a room, or activate Draw Wall.</p>
                </div>
              )}
            </aside>
          </div>

          <div className="ahg-floor-save-row">
            <div className="ahg-floor-feedback" aria-live="polite">
              {saveMessage && (
                <span className="ahg-floor-success">
                  <CheckCircle2 size={16} />
                  {saveMessage}
                </span>
              )}
              {saveError && (
                <span className="ahg-floor-error">
                  <AlertCircle size={16} />
                  {saveError}
                </span>
              )}
            </div>

            <button
              type="button"
              className="ahg-floor-save-button"
              onClick={handleSave}
              disabled={saving || loading}
            >
              {saving ? (
                <LoaderCircle className="ahg-spin" size={17} />
              ) : (
                <Save size={17} />
              )}
              {saving ? "Saving..." : "Save Floor Plans"}
            </button>
          </div>
        </>
      )}
    </section>
  );
}