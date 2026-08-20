import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({
  subsets: ["latin", "cyrillic"],
  variable: "--font-inter",
});

export const metadata: Metadata = {
  title: "nexus-mini",
  description:
    "Татаад ажиллуулаад, дээр нь модулиа бичээд, store-д нийтэлдэг платформын цөм",
};

// Театрчлахгүйгээр dark/light-ээ эрт тогтооно (flash-гүй).
const themeScript = `
try {
  var t = localStorage.getItem('nexus_theme');
  if (t === 'dark' || (t !== 'light' && matchMedia('(prefers-color-scheme: dark)').matches)) {
    document.documentElement.dataset.theme = 'dark';
  }
} catch (e) {}
`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="mn" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className={inter.variable}>{children}</body>
    </html>
  );
}
