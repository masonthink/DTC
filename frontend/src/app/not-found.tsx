import Link from "next/link";

export default function NotFound() {
  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center px-4 text-center">
      {/* Large numeral — typographic statement */}
      <p className="text-[88px] font-bold text-foreground/8 leading-none select-none mb-2 tabular-nums">
        404
      </p>
      <h1 className="text-[20px] font-bold text-foreground mb-2">页面不存在</h1>
      <p className="text-muted-foreground text-[13px] mb-8 max-w-xs leading-relaxed">
        你访问的页面已被移除或从未存在过
      </p>
      <Link
        href="/"
        className="bg-primary hover:bg-primary/90 active:scale-95 text-primary-foreground px-6 py-3 rounded-xl font-semibold text-[13px] transition-all duration-150 shadow-primary-sm"
      >
        回到首页
      </Link>
    </div>
  );
}
