import type { CalculateResponse } from "@/lib/types";

interface Props {
  result: CalculateResponse | null;
  order: number | null;
}

function format(n: number): string {
  return n.toLocaleString("en-US");
}

export function ResultPanel({ result, order }: Props) {
  if (!result) {
    return (
      <div className="rounded-xl border border-dashed border-zinc-300 px-4 py-10 text-center text-sm text-zinc-400 dark:border-zinc-700 dark:text-zinc-600">
        Run a calculation to see the breakdown.
      </div>
    );
  }

  if (result.total_packs === 0) {
    return (
      <div className="rounded-xl bg-zinc-100 px-4 py-6 text-center text-sm text-zinc-500 dark:bg-zinc-800/50 dark:text-zinc-400">
        Order is zero — no packs needed.
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-200 bg-zinc-50/60 p-5 dark:border-zinc-800 dark:bg-zinc-950/40">
      <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1 font-mono text-lg">
        {result.packs.map((p, i) => (
          <span key={p.size} className="inline-flex items-center">
            {i > 0 && <span className="mx-2 text-zinc-300 dark:text-zinc-700">+</span>}
            <span className="rounded-md bg-amber-500/10 px-2 py-1 text-amber-700 ring-1 ring-amber-500/30 dark:text-amber-300">
              <span className="font-semibold">{p.count}</span>
              <span className="text-amber-700/60 dark:text-amber-300/60"> × </span>
              <span className="tabular-nums">{format(p.size)}</span>
            </span>
          </span>
        ))}
      </div>

      <dl className="mt-5 grid grid-cols-3 gap-4 border-t border-zinc-200 pt-4 text-sm dark:border-zinc-800">
        <Stat label="Total items" value={format(result.total_items)} />
        <Stat label="Total packs" value={format(result.total_packs)} />
        <Stat
          label="Overshoot"
          value={format(result.overshoot)}
          hint={
            order !== null && order > 0
              ? `vs order of ${format(order)}`
              : undefined
          }
          accent={result.overshoot === 0 ? "ok" : "warn"}
        />
      </dl>
    </div>
  );
}

interface StatProps {
  label: string;
  value: string;
  hint?: string;
  accent?: "ok" | "warn";
}

function Stat({ label, value, hint, accent }: StatProps) {
  const tone =
    accent === "ok"
      ? "text-emerald-600 dark:text-emerald-400"
      : accent === "warn"
        ? "text-amber-600 dark:text-amber-400"
        : "text-zinc-900 dark:text-zinc-100";
  return (
    <div>
      <dt className="text-xs uppercase tracking-wider text-zinc-500 dark:text-zinc-500">{label}</dt>
      <dd className={`mt-1 font-mono text-xl font-semibold tabular-nums ${tone}`}>{value}</dd>
      {hint ? <dd className="mt-0.5 text-xs text-zinc-400 dark:text-zinc-500">{hint}</dd> : null}
    </div>
  );
}
