# Concors Frontend Design System

## Project
- Path: `/Users/mason/Documents/digital-twin-community/frontend/`
- Framework: Next.js 14+ App Router, Tailwind CSS, shadcn/ui tokens
- Language: Chinese (CJK typography required)
- App name: Concors (数字分身社区)

## Design Tokens (globals.css)
- Primary: `hsl(243 75% 59%)` — violet-indigo
- Background: `hsl(220 20% 98%)` — warm off-white
- All colors use CSS custom properties via HSL: `--primary`, `--background`, `--foreground`, `--card`, `--muted`, `--border`
- Primary shadows: `--shadow-primary-sm`, `--shadow-primary-md` (purple-tinted)
- Dark mode: primary lightens to `hsl(243 72% 65%)`

## Color Token Conventions
- NEVER use hardcoded `slate-*`, `indigo-*` Tailwind classes in components
- Use semantic tokens: `text-foreground`, `text-muted-foreground`, `bg-card`, `bg-background`, `bg-muted`, `border-border`, `text-primary`, `bg-primary`
- Status colors use solid Tailwind (not opacity): `bg-amber-50 text-amber-600 border-amber-100`, `bg-emerald-50 text-emerald-600 border-emerald-100`, `bg-red-50 text-red-600 border-red-100`
- Selected/active state: `bg-primary/8 border-primary text-primary`
- Muted/inactive card: `bg-muted/60 border-border text-muted-foreground`

## Typography
- Font stack: Inter + PingFang SC + Hiragino Sans GB + Microsoft YaHei
- CJK line-height: 1.75 (body), set in globals.css
- Font sizes: explicit px values — `text-[11px]`, `text-[12px]`, `text-[13px]`, `text-[14px]`, `text-[15px]`, `text-[18px]`
- Section labels: `text-[11px] font-semibold text-muted-foreground uppercase tracking-wider`
- Error messages: `text-red-500 text-[11px]`

## Component Patterns
- Cards: `bg-card border border-border rounded-2xl p-5 shadow-xs`
- Header (sticky): `bg-card/92 backdrop-blur-2xl border-b border-border/60`
- Back button: `w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150`
- Primary CTA button: `bg-primary hover:bg-primary/90 text-primary-foreground font-semibold py-4 rounded-2xl shadow-primary-md transition-all duration-150 active:scale-[0.98]`
- Form inputs: `bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[13px]`
- Gradient bottom fade: `bg-gradient-to-t from-background via-background/95 to-transparent`
- Tags/pills (selected): `border-primary bg-primary/8 text-primary`
- Tags/pills (unselected): `border-border bg-card text-muted-foreground hover:border-primary/30`

## Utility Classes (globals.css)
- `.bg-mesh` — radial gradient background with primary color hints
- `.text-gradient-primary` — gradient text from primary to violet
- `.glass` — glassmorphism card
- `.card-elevated` — elevated card with primary shadow
- `.shadow-primary-sm` / `.shadow-primary-md` — Tailwind shadow tokens from CSS vars

## Toaster
- Normal: `bg-foreground border-foreground/10 text-background`
- Destructive: `bg-red-600 border-red-500/50 text-white`

## Linter Behavior
- A background linter modifies files between Read and Write/Edit operations
- Always re-read files immediately before editing
- Use Write (full rewrite) when linter interference causes string-match failures

## Status Badge Patterns (TopicStatusBadge)
Solid Tailwind colors, not opacity modifiers:
- pending/matching: `bg-amber-50 text-amber-600 border-amber-200/80 pulse: true`
- matched: `bg-blue-50 text-blue-600 border-blue-200/80`
- discussion_active: `bg-primary/8 text-primary border-primary/20 pulse: true`
- report_generating: `bg-violet-50 text-violet-600 border-violet-200/80 pulse: true`
- completed: `bg-emerald-50 text-emerald-600 border-emerald-200/80`
