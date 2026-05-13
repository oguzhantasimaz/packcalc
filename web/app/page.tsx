import { Calculator } from "@/components/Calculator";
import { PackEditor } from "@/components/PackEditor";

export default function Home() {
  const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");
  const docsUrl = `${apiBase}/docs/`;

  return (
    <main className="mx-auto max-w-6xl px-6 py-10 md:py-16">
      <header className="mb-10 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-amber-600 dark:text-amber-400">
            <span className="inline-block h-1.5 w-1.5 rounded-full bg-amber-500" />
            <span>Pack Calculator</span>
          </div>
          <h1 className="mt-2 text-4xl font-semibold tracking-tight">
            Ship whole packs. <span className="text-zinc-400 dark:text-zinc-500">Minimize waste.</span>
          </h1>
          <p className="mt-2 max-w-xl text-sm text-zinc-500 dark:text-zinc-400">
            Configure pack sizes on the left. Calculate an order on the right. Rule 2 (minimum items) takes
            precedence over rule 3 (minimum packs).
          </p>
        </div>
        <nav className="flex flex-wrap items-center gap-4 text-sm">
          <a
            href={docsUrl}
            target="_blank"
            rel="noreferrer"
            className="text-zinc-500 transition hover:text-amber-600 dark:text-zinc-400 dark:hover:text-amber-400"
          >
            API docs ↗
          </a>
          <a
            href="https://github.com/oguzhantasimaz/packcalc"
            target="_blank"
            rel="noreferrer"
            className="text-zinc-500 transition hover:text-amber-600 dark:text-zinc-400 dark:hover:text-amber-400"
          >
            GitHub ↗
          </a>
        </nav>
      </header>

      <div className="grid gap-6 md:grid-cols-2">
        <PackEditor />
        <Calculator />
      </div>

      <footer className="mt-16 text-center text-xs text-zinc-400 dark:text-zinc-600">
        Built with Go (Fiber) + Next.js. Spec at{" "}
        <a href={docsUrl} target="_blank" rel="noreferrer" className="underline-offset-2 hover:underline">
          /docs
        </a>
        .
      </footer>
    </main>
  );
}
