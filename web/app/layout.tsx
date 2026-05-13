import type { Metadata } from "next";
import { Toaster } from "sonner";
import "./globals.css";

export const metadata: Metadata = {
  title: "Packcalc",
  description: "Whole-pack shipment calculator: minimum items, then minimum packs.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-zinc-50 text-zinc-900 antialiased dark:bg-zinc-950 dark:text-zinc-100">
        <div className="bg-grid min-h-screen">{children}</div>
        <Toaster richColors position="top-right" closeButton />
      </body>
    </html>
  );
}
