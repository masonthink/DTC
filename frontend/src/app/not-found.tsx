import Link from "next/link";

export default function NotFound() {
  return (
    <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center px-4 text-center">
      <p className="text-6xl mb-6">404</p>
      <h1 className="text-2xl font-bold text-white mb-3">页面不存在</h1>
      <p className="text-slate-400 text-sm mb-8 max-w-xs">
        你访问的页面已被移除或从未存在过
      </p>
      <Link
        href="/dashboard"
        className="bg-indigo-600 hover:bg-indigo-500 active:scale-95 text-white px-6 py-3 rounded-xl font-medium transition-all"
      >
        回到首页
      </Link>
    </div>
  );
}
