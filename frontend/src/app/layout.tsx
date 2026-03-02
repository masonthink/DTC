import type { Metadata, Viewport } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Toaster } from "@/components/ui/toaster";
import { SWRProvider } from "@/components/providers/SWRProvider";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
  weight: ["300", "400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "Concors - AI 多角度讨论，帮你做更好的决策",
  description: "提交一个问题，4 位 AI 从不同角度深度讨论，生成专属分析报告。发现志同道合的人。",
  manifest: "/manifest.json",
  icons: {
    icon: "/favicon.ico",
    apple: "/apple-touch-icon.png",
  },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  themeColor: "#f5f5f8",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning className={inter.variable}>
      <body className="bg-background text-foreground antialiased font-sans">
        <SWRProvider>
          {children}
        </SWRProvider>
        <Toaster />
      </body>
    </html>
  );
}
