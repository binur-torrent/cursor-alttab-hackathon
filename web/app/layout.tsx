import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "LumiCity AI — Istanbul streetlight intelligence",
  description:
    "AI-driven urban lighting analysis and adaptive-lighting simulation for Istanbul. Detects streetlight fixtures from street imagery and scores segments for safety risk.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-slate-950 text-slate-100 antialiased">
        {children}
      </body>
    </html>
  );
}
