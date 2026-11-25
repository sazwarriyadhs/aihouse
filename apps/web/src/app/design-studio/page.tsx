import Image from "next/image";
import Link from "next/link";
import DashboardShell from "@/components/dashboard-shell";
import { ArrowLeft, ArrowRight, CheckCircle2 } from "lucide-react";
import { houseStyles } from "@/lib/house-styles";

export default function DesignStudioPage() {
  return (
    <DashboardShell activeItem="AI Design Studio" breadcrumb="Design Studio">
      <main className="ahg-design-studio">
        <div className="ahg-design-studio-heading">
          <div>
            <span className="ahg-design-studio-eyebrow">AI HOUSE GENERATOR / DESIGN STUDIO</span>
            <h1>Explore House Styles</h1>
            <p>Choose an architectural direction to start your house project.</p>
          </div>
          <Link href="/" className="ahg-design-studio-back">
            <ArrowLeft size={16} /> Back to Dashboard
          </Link>
        </div>

        <div className="ahg-design-studio-summary">
          <div>
            <strong>{houseStyles.length} architectural styles</strong>
            <span>Browse the collection and select a style for your next project.</span>
          </div>
          <span className="ahg-design-studio-count">25 STYLES</span>
        </div>

        <section className="ahg-house-style-grid" aria-label="House style catalog">
          {houseStyles.map((style, index) => (
            <article className="ahg-house-style-card" key={style.id}>
              <Link
                href={`/projects/new?style=${encodeURIComponent(style.id)}`}
                className="ahg-house-style-image-link"
                aria-label={`Select ${style.name}`}
              >
                <div className="ahg-house-style-image">
                  <Image
                    src={style.image}
                    alt={`${style.name} house architecture`}
                    fill
                    sizes="(max-width: 640px) 100vw, (max-width: 1100px) 50vw, 33vw"
                  />
                  <span className="ahg-house-style-number">{String(index + 1).padStart(2, "0")}</span>
                </div>
              </Link>
              <div className="ahg-house-style-copy">
                <div>
                  <h2>{style.name}</h2>
                  <p>{style.description}</p>
                </div>
                <Link
                  href={`/projects/new?style=${encodeURIComponent(style.id)}`}
                  className="ahg-house-style-select"
                >
                  <CheckCircle2 size={15} /> Use This Style <ArrowRight size={15} />
                </Link>
              </div>
            </article>
          ))}
        </section>
      </main>
    </DashboardShell>
  );
}
