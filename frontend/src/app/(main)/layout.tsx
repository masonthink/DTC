import { BottomNav } from "@/components/layout/BottomNav";

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-slate-950">
      <main className="pb-24 max-w-screen-sm mx-auto">{children}</main>
      <BottomNav />
    </div>
  );
}
