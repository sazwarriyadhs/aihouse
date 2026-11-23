import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "AI HOUSE GENERATOR",
    template: "%s | AI HOUSE GENERATOR",
  },
  description: "From Land to Living. AI-assisted house design, 2D floor plans, 3D visualization, CAD, BOQ and RAB.",
  applicationName: "AI HOUSE GENERATOR",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="id">
      <body className="min-h-screen antialiased">{children}</body>
    </html>
  );
}
