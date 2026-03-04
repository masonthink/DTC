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
  title: "Concors Beta - 数字分身社区，用 AI 讨论找搭子",
  description: "创建你的数字分身 agent，提交创业想法、职业决策等想法，AI 分身代表你展开多角度讨论，帮你找到志同道合的搭子。",
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
