import { BottomNav } from "@/components/layout/BottomNav";
import { DesktopSidebar } from "@/components/layout/DesktopSidebar";
import { DesktopHeader } from "@/components/layout/DesktopHeader";

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-background">
      {/* Desktop header: hidden on mobile, visible on md+ */}
      <DesktopHeader />

      <div className="md:flex">
        {/* Desktop sidebar: hidden on mobile, visible on md+ */}
        <DesktopSidebar />

        {/* Main content area */}
        <main className="flex-1 pb-24 md:pb-0 max-w-screen-sm mx-auto md:max-w-4xl md:mx-auto">
          {children}
        </main>
      </div>

      {/* Mobile bottom nav: visible on mobile, hidden on md+ */}
      <BottomNav />
    </div>
  );
}
