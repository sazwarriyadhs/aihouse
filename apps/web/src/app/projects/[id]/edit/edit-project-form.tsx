"use client";

import { useState, type FormEvent } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  ArrowRight,
  House,
  LoaderCircle,
  Save,
} from "lucide-react";
import {
  updateProject,
  type CreateProjectInput,
  type Project,
} from "@/lib/api/projects";
import { houseStyles } from "@/lib/house-styles";
import DashboardShell from "@/components/dashboard-shell";

interface EditProjectFormProps {
  project: Project;
  initialValues: CreateProjectInput;
  initialStyleId: string;
}

const orientations = [
  ["N", "North"],
  ["NE", "Northeast"],
  ["E", "East"],
  ["SE", "Southeast"],
  ["S", "South"],
  ["SW", "Southwest"],
  ["W", "West"],
  ["NW", "Northwest"],
];

export default function EditProjectForm({
  project,
  initialValues,
  initialStyleId,
}: EditProjectFormProps) {
  const router = useRouter();
  const [form, setForm] = useState<CreateProjectInput>(initialValues);
  const [styleId, setStyleId] = useState(initialStyleId);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const selectedStyle =
    houseStyles.find((style) => style.id === styleId) ?? null;

  const area = form.land_width_m * form.land_depth_m;

  function update<K extends keyof CreateProjectInput>(
    key: K,
    value: CreateProjectInput[K],
  ) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    if (!form.name.trim()) {
      setError("Project name is required.");
      return;
    }

    if (
      !Number.isFinite(form.land_width_m) ||
      !Number.isFinite(form.land_depth_m) ||
      form.land_width_m < 1 ||
      form.land_width_m > 1000 ||
      form.land_depth_m < 1 ||
      form.land_depth_m > 1000
    ) {
      setError("Land dimensions must be between 1 and 1,000 metres.");
      return;
    }

    if (![1, 2].includes(form.floors)) {
      setError("The number of floors must be 1 or 2.");
      return;
    }

    setSubmitting(true);

    const result = await updateProject(project.id, {
      ...form,
      name: form.name.trim(),
      description: form.description.trim(),
      location: form.location.trim(),
      ...(selectedStyle
        ? {
            design_style_id: selectedStyle.id,
            design_style_name: selectedStyle.name,
            design_style_image: selectedStyle.image,
          }
        : {}),
    });

    setSubmitting(false);

    if (result.error || !result.data) {
      setError(result.error || "Could not update project.");
      return;
    }

    router.push(`/projects/${project.id}`);
    router.refresh();
  }

  return (
    <DashboardShell activeItem="My Projects" breadcrumb="Edit Project">
      <main className="project-create-page">
        <div className="project-create-container">
          <Link
            href={`/projects/${project.id}`}
            className="project-back-link"
          >
            <ArrowLeft size={17} />
            Back to Project
          </Link>

          <div className="project-create-heading">
            <span className="project-create-icon">
              <House size={23} />
            </span>
            <p className="eyebrow">EDIT PROJECT</p>
            <h1>Refine your home project.</h1>
            <p>
              Update your project details. Saving changes creates a new
              design version while preserving previous versions.
            </p>
          </div>

          <form className="project-form" onSubmit={handleSubmit}>
            <section className="project-form-section">
              <div className="project-form-section-heading">
                <span>01</span>
                <div>
                  <h2>Project information</h2>
                  <p>Update the identity and location of your project.</p>
                </div>
              </div>

              <label className="project-field">
                <span>Project name <b>*</b></span>
                <input
                  required
                  maxLength={200}
                  value={form.name}
                  onChange={(event) => update("name", event.target.value)}
                />
              </label>

              <label className="project-field">
                <span>Project location</span>
                <input
                  maxLength={300}
                  value={form.location}
                  onChange={(event) => update("location", event.target.value)}
                />
              </label>

              <label className="project-field">
                <span>Description</span>
                <textarea
                  rows={3}
                  maxLength={5000}
                  value={form.description}
                  onChange={(event) =>
                    update("description", event.target.value)
                  }
                />
              </label>
            </section>

            <section className="project-form-section">
              <div className="project-form-section-heading">
                <span>02</span>
                <div>
                  <h2>Land dimensions</h2>
                  <p>Adjust the land area and its orientation.</p>
                </div>
              </div>

              <div className="project-form-two-columns">
                <label className="project-field">
                  <span>Width (metres) <b>*</b></span>
                  <input
                    type="number"
                    required
                    min={1}
                    max={1000}
                    step="0.01"
                    value={form.land_width_m}
                    onChange={(event) =>
                      update("land_width_m", Number(event.target.value))
                    }
                  />
                </label>

                <label className="project-field">
                  <span>Depth (metres) <b>*</b></span>
                  <input
                    type="number"
                    required
                    min={1}
                    max={1000}
                    step="0.01"
                    value={form.land_depth_m}
                    onChange={(event) =>
                      update("land_depth_m", Number(event.target.value))
                    }
                  />
                </label>
              </div>

              <div className="project-area-preview">
                <span>Calculated land area</span>
                <strong>
                  {Number.isFinite(area)
                    ? area.toLocaleString("en-US")
                    : "—"}{" "}
                  m²
                </strong>
              </div>

              <label className="project-field">
                <span>Land orientation</span>
                <select
                  value={form.orientation}
                  onChange={(event) =>
                    update("orientation", event.target.value)
                  }
                >
                  {orientations.map(([value, label]) => (
                    <option value={value} key={value}>
                      {label} ({value})
                    </option>
                  ))}
                </select>
              </label>
            </section>

            <section className="project-form-section">
              <div className="project-form-section-heading">
                <span>03</span>
                <div>
                  <h2>Building requirements</h2>
                  <p>Update the initial number of floors.</p>
                </div>
              </div>

              <div className="project-floor-options">
                {[1, 2].map((floor) => (
                  <button
                    type="button"
                    key={floor}
                    className={`project-floor-option ${
                      form.floors === floor ? "selected" : ""
                    }`}
                    onClick={() => update("floors", floor)}
                    aria-pressed={form.floors === floor}
                  >
                    <House size={21} />
                    <strong>
                      {floor} {floor === 1 ? "Floor" : "Floors"}
                    </strong>
                    <small>
                      {floor === 1
                        ? "Single-storey home"
                        : "Two-storey home"}
                    </small>
                  </button>
                ))}
              </div>
            </section>

            <section className="project-form-section">
              <div className="project-form-section-heading">
                <span>04</span>
                <div>
                  <h2>Architectural style</h2>
                  <p>
                    Keep the current style or choose a different one.
                  </p>
                </div>
              </div>

              <label className="project-field">
                <span>House style</span>
                <select
                  value={styleId}
                  onChange={(event) => setStyleId(event.target.value)}
                >
                  <option value="">Keep existing style if available</option>
                  {houseStyles.map((style) => (
                    <option value={style.id} key={style.id}>
                      {style.name}
                    </option>
                  ))}
                </select>
              </label>

              {selectedStyle && (
                <div className="project-selected-style">
                  <div className="project-selected-style-image">
                    <Image
                      src={selectedStyle.image}
                      alt={`${selectedStyle.name} house style`}
                      fill
                      sizes="112px"
                    />
                  </div>
                  <div className="project-selected-style-copy">
                    <span>SELECTED HOUSE STYLE</span>
                    <strong>{selectedStyle.name}</strong>
                    <p>{selectedStyle.description}</p>
                  </div>
                </div>
              )}
            </section>

            {error && (
              <div className="project-form-error" role="alert">
                {error}
              </div>
            )}

            <div className="project-form-actions">
              <Link
                href={`/projects/${project.id}`}
                className="project-cancel-button"
              >
                Cancel
              </Link>
              <button
                type="submit"
                className="primary-button"
                disabled={submitting}
              >
                {submitting ? (
                  <>
                    <LoaderCircle
                      size={17}
                      className="project-spinner"
                    />
                    Saving...
                  </>
                ) : (
                  <>
                    Save Changes
                    <Save size={17} />
                    <ArrowRight size={17} />
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      </main>
    </DashboardShell>
  );
}
