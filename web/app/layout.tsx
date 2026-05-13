import type { Metadata } from "next";
import { Instrument_Serif, JetBrains_Mono, Manrope } from "next/font/google";
import { Toaster } from "sonner";

import { Background } from "@/components/Background";
import "./globals.css";

// Display: an editorial serif we use sparingly for emphasis (italic on
// the wordmark, occasional accents). It buys the page a bit of gravity.
const instrumentSerif = Instrument_Serif({
  subsets: ["latin"],
  weight: ["400"],
  style: ["normal", "italic"],
  variable: "--font-display",
  display: "swap",
});

// Body: Manrope. Friendlier than Inter, rounder terminals, geometric
// without feeling generic.
const manrope = Manrope({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-sans",
  display: "swap",
});

// Mono: JetBrains Mono for tabular data. Excellent at sizes 12-20px,
// distinct number glyphs, well-suited to a calculator UI.
const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-mono",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Packcalc — whole-pack shipment calculator",
  description:
    "Compute the optimal whole-pack shipment for an order. Minimum items, then minimum packs.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      className={`${instrumentSerif.variable} ${manrope.variable} ${jetbrainsMono.variable}`}
    >
      {/*
        suppressHydrationWarning is needed on <body> because browser
        extensions (ColorZilla, Grammarly, LastPass, etc.) inject
        attributes like cz-shortcut-listen before React hydrates. The
        warning is harmless but spammy in dev. This does NOT suppress
        warnings on any descendant.
      */}
      <body className="font-sans antialiased" suppressHydrationWarning>
        <Background />
        {children}
        <Toaster
          richColors
          position="top-right"
          closeButton
          theme="dark"
          toastOptions={{
            style: {
              background: "rgba(12, 19, 32, 0.92)",
              border: "1px solid rgba(0,172,222,0.22)",
              color: "#DCE6F2",
            },
          }}
        />
      </body>
    </html>
  );
}
