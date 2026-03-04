import { BottomNav } from "@/components/layout/BottomNav";
import { DesktopSidebar } from "@/components/layout/DesktopSidebar";

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-background md:flex">
      <DesktopSidebar />
      <main className="flex-1 pb-24 md:pb-0 max-w-screen-sm mx-auto md:max-w-4xl">
        {children}
      </main>
      <BottomNav />
    </div>
  );
}
