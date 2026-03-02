import Link from "next/link";

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Nav */}
      <header className="sticky top-0 z-50 bg-white/90 backdrop-blur-xl border-b border-slate-100">
        <div className="max-w-screen-sm mx-auto px-5 py-4 flex items-center justify-between">
          <h1 className="text-lg font-bold text-slate-900">
            <span className="text-indigo-600">C</span>oncors
          </h1>
          <div className="flex items-center gap-3">
            <Link
              href="/login"
              className="text-sm text-slate-600 hover:text-slate-900 transition-colors"
            >
              登录
            </Link>
            <Link
              href="/register"
              className="text-sm bg-indigo-600 hover:bg-indigo-500 text-white px-4 py-2 rounded-xl font-medium transition-all active:scale-95"
            >
              免费注册
            </Link>
          </div>
        </div>
      </header>

      <main className="max-w-screen-sm mx-auto px-5">
        {/* Hero */}
        <section className="pt-16 pb-12 text-center">
          <div className="inline-flex items-center gap-2 bg-indigo-50 border border-indigo-100 text-indigo-600 text-xs font-medium px-3 py-1.5 rounded-full mb-6">
            <span className="w-1.5 h-1.5 bg-indigo-500 rounded-full animate-pulse" />
            AI 驱动的深度思维碰撞
          </div>
          <h2 className="text-3xl font-bold text-slate-900 leading-snug mb-4">
            让你的数字分身
            <br />
            <span className="text-indigo-600">帮你做更好的决策</span>
          </h2>
          <p className="text-slate-500 text-base leading-relaxed mb-8 max-w-xs mx-auto">
            提交一个话题，AI 分身们从不同视角展开深度讨论，48小时内生成专属分析报告
          </p>
          <Link
            href="/register"
            className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white font-semibold px-8 py-4 rounded-2xl text-base transition-all active:scale-[0.98] shadow-lg shadow-indigo-600/20"
          >
            开始使用 →
          </Link>
          <p className="text-xs text-slate-400 mt-4">免费 · 无需下载 · 1分钟注册</p>
        </section>

        {/* How it works */}
        <section className="pb-12">
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider text-center mb-6">
            如何运作
          </p>
          <div className="space-y-4">
            {[
              {
                step: "01",
                emoji: "🤖",
                title: "创建你的数字分身",
                desc: "填写你的专业背景、思维方式和行业经验，生成一个代表你的 AI 分身",
              },
              {
                step: "02",
                emoji: "💬",
                title: "提交你的话题",
                desc: "描述你正在思考的问题——商业方向、职业决策、技术选型，都可以",
              },
              {
                step: "03",
                emoji: "⚡",
                title: "AI 分身展开讨论",
                desc: "系统为你匹配 4 位不同角色的 AI 分身，进行 4 轮深度的多角度辩论",
              },
              {
                step: "04",
                emoji: "📊",
                title: "获得专属报告",
                desc: "收到包含共识、分歧、盲点和行动建议的完整分析报告",
              },
            ].map((item) => (
              <div
                key={item.step}
                className="flex items-start gap-4 bg-slate-50 rounded-2xl p-5"
              >
                <div className="flex-shrink-0 w-10 h-10 bg-white border border-slate-200 rounded-xl flex items-center justify-center text-xl shadow-sm">
                  {item.emoji}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs font-bold text-indigo-400 font-mono">{item.step}</span>
                    <h3 className="text-sm font-semibold text-slate-900">{item.title}</h3>
                  </div>
                  <p className="text-xs text-slate-500 leading-relaxed">{item.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* Features */}
        <section className="pb-12">
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider text-center mb-6">
            核心特性
          </p>
          <div className="grid grid-cols-2 gap-3">
            {[
              { emoji: "🔒", title: "匿名保护", desc: "你的真实身份完全匿名，分身用随机 ID 参与讨论" },
              { emoji: "🎭", title: "多角色辩论", desc: "质疑者、支持者、补充者、探究者四种视角" },
              { emoji: "🧠", title: "深度分析", desc: "4 轮递进式讨论，比单次 AI 问答深入 10 倍" },
              { emoji: "🤝", title: "真实连接", desc: "发现志同道合的人，在匿名保护下建立真实连接" },
            ].map((f) => (
              <div key={f.title} className="bg-slate-50 rounded-2xl p-4">
                <div className="text-2xl mb-2">{f.emoji}</div>
                <h3 className="text-sm font-semibold text-slate-900 mb-1">{f.title}</h3>
                <p className="text-xs text-slate-500 leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Use cases */}
        <section className="pb-12">
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider text-center mb-6">
            适合什么场景
          </p>
          <div className="flex flex-wrap gap-2 justify-center">
            {[
              "💡 创业方向验证",
              "💼 跳槽/转行决策",
              "⚙️ 技术架构选型",
              "🎨 产品功能评估",
              "📈 投资项目分析",
              "🌐 商业战略规划",
            ].map((tag) => (
              <span
                key={tag}
                className="text-sm text-slate-700 bg-slate-100 border border-slate-200 px-4 py-2 rounded-full"
              >
                {tag}
              </span>
            ))}
          </div>
        </section>

        {/* CTA */}
        <section className="pb-16 text-center">
          <div className="bg-gradient-to-br from-indigo-50 to-purple-50 border border-indigo-100 rounded-3xl p-8">
            <div className="text-4xl mb-4">🚀</div>
            <h3 className="text-xl font-bold text-slate-900 mb-2">
              开始你的第一次深度思维碰撞
            </h3>
            <p className="text-slate-500 text-sm mb-6 leading-relaxed">
              免费注册，创建你的数字分身，提交你最想突破的问题
            </p>
            <Link
              href="/register"
              className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white font-semibold px-8 py-4 rounded-2xl transition-all active:scale-[0.98] shadow-lg shadow-indigo-600/20"
            >
              立即免费注册
            </Link>
            <p className="text-xs text-slate-400 mt-4">
              已有账号？
              <Link href="/login" className="text-indigo-500 hover:text-indigo-600 ml-1">
                直接登录
              </Link>
            </p>
          </div>
        </section>
      </main>

      <footer className="border-t border-slate-100 py-6 text-center">
        <p className="text-xs text-slate-400">© 2025 Concors · 数字分身社区</p>
      </footer>
    </div>
  );
}
