import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "AI Software Factory",
  description: "Full-stack AI-powered software development platform",
};

// `useSearchParams` is called by `useProjectFilters` (and therefore
// every page that goes through ProjectPickerGate). The Next.js
// static-prerender pass requires a Suspense boundary or a
// `force-dynamic` opt-out for any client component that calls it.
// Whole-app dynamic rendering is the simpler choice for this
// dashboard — there is no static-export goal here.
export const dynamic = "force-dynamic";

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      data-theme="dark"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      suppressHydrationWarning
    >
      <body className="min-h-full bg-gray-900 text-gray-100">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
