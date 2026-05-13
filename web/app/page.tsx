import { Calculator } from "@/components/Calculator";
import { PackEditor } from "@/components/PackEditor";

export default function Home() {
  const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");
  const docsUrl = `${apiBase}/docs/`;

  return (
    <main className="relative mx-auto max-w-6xl px-6 pb-20 pt-12 md:pt-20">
      <header className="mb-14 flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
        <div className="animate-fade-up">
          <div className="pill">
            <span className="relative inline-flex h-1.5 w-1.5">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-brand-400 opacity-60" />
              <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-brand-400" />
            </span>
            Pack Calculator
          </div>

          <h1 className="mt-4 font-display text-5xl leading-[1.05] tracking-tight text-ink-50 md:text-6xl">
            Ship whole packs.
            <br />
            <span className="font-display italic text-brand-400">Minimize waste.</span>
          </h1>

          <p className="mt-4 max-w-xl text-sm leading-relaxed text-ink-300">
            Configure pack sizes on the left, calculate an order on the right. Rule 2 (minimum items)
            takes precedence over rule 3 (minimum packs). The algorithm is two-phase DP — robust to
            adversarial pack sets like
            <span className="mx-1 rounded bg-ink-800 px-1.5 py-0.5 font-mono text-[0.78em] text-brand-200">
              [23, 31, 53]
            </span>
            where greedy fails.
          </p>
        </div>

        <nav className="flex shrink-0 flex-wrap items-center gap-3 text-xs uppercase tracking-[0.16em] text-ink-300">
          <a
            href={docsUrl}
            target="_blank"
            rel="noreferrer"
            className="group inline-flex items-center gap-1.5 rounded-md border border-ink-700/60 bg-ink-900/40 px-3 py-2 transition hover:border-brand-400/40 hover:text-brand-300"
          >
            <span>API · /docs</span>
            <Arrow />
          </a>
          <a
            href="https://github.com/oguzhantasimaz/packcalc"
            target="_blank"
            rel="noreferrer"
            className="group inline-flex items-center gap-1.5 rounded-md border border-ink-700/60 bg-ink-900/40 px-3 py-2 transition hover:border-brand-400/40 hover:text-brand-300"
          >
            <span>GitHub</span>
            <Arrow />
          </a>
        </nav>
      </header>

      <div className="grid gap-6 md:grid-cols-2 [&>*:nth-child(1)]:[animation-delay:120ms] [&>*:nth-child(2)]:[animation-delay:240ms]">
        <PackEditor />
        <Calculator />
      </div>

      <footer className="mt-20 flex flex-wrap items-center justify-between gap-2 border-t border-ink-700/40 pt-6 text-[11px] uppercase tracking-[0.18em] text-ink-400">
        <span>Go · Fiber · Next.js · Tailwind</span>
        <a
          href={docsUrl}
          target="_blank"
          rel="noreferrer"
          className="transition hover:text-brand-300"
        >
          OpenAPI 3.1 — view spec
        </a>
      </footer>
    </main>
  );
}

function Arrow() {
  return (
    <svg
      width="10"
      height="10"
      viewBox="0 0 10 10"
      fill="none"
      className="transition group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
    >
      <path d="M2 8 L8 2 M8 2 H3.5 M8 2 V6.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="square" />
    </svg>
  );
}
