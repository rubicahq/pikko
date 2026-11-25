import type { Metadata } from "next";
import { Inter, Plus_Jakarta_Sans } from "next/font/google";
import { MARKETING_DEFAULT_DESCRIPTION, marketingURL } from "@/lib/marketing-seo";
import "./globals.css";

/* Inter kept loaded (--font-inter / font-inter) so existing consumers do not break abruptly */
const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

/* Plus Jakarta Sans is the product + marketing face (font-sans / font-display) */
const jakarta = Plus_Jakarta_Sans({
  subsets: ["latin"],
  variable: "--font-jakarta",
  display: "swap",
  weight: ["400", "500", "600"],
});

const appEnv = process.env.NEXT_PUBLIC_APP_ENV || process.env.NODE_ENV || "development";
const isProdPublic = appEnv === "production";

export const metadata: Metadata = {
  metadataBase: new URL(marketingURL("/")),
  title: {
    default: "Pikko",
    template: "%s · Pikko",
  },
  description: MARKETING_DEFAULT_DESCRIPTION,
  applicationName: "Pikko",
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "any" },
      { url: "/brand/pikko-icon.png", type: "image/png", sizes: "192x192" },
    ],
    apple: [{ url: "/brand/apple-touch-icon.png", sizes: "180x180" }],
  },
  openGraph: {
    type: "website",
    siteName: "Pikko",
    locale: "id_ID",
    title: "Pikko",
    description: MARKETING_DEFAULT_DESCRIPTION,
    url: marketingURL("/"),
  },
  twitter: {
    card: "summary_large_image",
    title: "Pikko",
    description: MARKETING_DEFAULT_DESCRIPTION,
  },
  robots: isProdPublic
    ? { index: true, follow: true }
    : { index: false, follow: false },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="id">
      <body className={`${inter.variable} ${jakarta.variable} font-sans antialiased`}>
        {children}
      </body>
    </html>
  );
}
