import type { CalculateResponse } from "@/lib/types";

interface Props {
  result: CalculateResponse | null;
  order: number | null;
}

function fmt(n: number): string {
  return n.toLocaleString("en-US");
}

export function ResultPanel({ result, order }: Props) {
  if (!result) {
    return (
      <div className="rounded-xl border border-dashed border-ink-700/40 bg-ink-900/30 px-4 py-12 text-center">
        <div className="font-mono text-[10px] uppercase tracking-[0.22em] text-ink-500">
          Awaiting input
        </div>
        <p className="mt-2 text-sm text-ink-400">Run a calculation to see the breakdown.</p>
      </div>
    );
  }

  if (result.total_packs === 0) {
    return (
      <div className="rounded-xl border border-ink-700/40 bg-ink-900/40 px-4 py-8 text-center text-sm text-ink-400">
        Order is zero — no packs needed.
      </div>
    );
  }

  return (
    <div className="relative overflow-hidden rounded-xl border border-brand-400/15 bg-ink-900/40 p-5">
      {/* faint cyan grid inside the panel for visual continuity */}
      <div
        className="pointer-events-none absolute inset-0 opacity-50"
        style={{
          backgroundImage:
            "radial-gradient(circle at 1px 1px, rgba(0,172,222,0.15) 1px, transparent 1.2px)",
          backgroundSize: "22px 22px",
        }}
      />

      <div className="relative">
        <div className="font-mono text-[10px] uppercase tracking-[0.22em] text-brand-300">
          Breakdown
        </div>

        <div className="mt-3 flex flex-wrap items-center gap-x-2 gap-y-2">
          {result.packs.map((p, i) => (
            <span key={p.size} className="inline-flex items-center">
              {i > 0 && <span className="mx-2 text-ink-600">+</span>}
              <span className="chip">
                <span className="font-mono text-base font-semibold text-brand-100 tabular-nums">
                  {fmt(p.count)}
                </span>
                <span className="font-mono text-xs text-brand-300/70">×</span>
                <span className="font-mono text-base font-semibold text-brand-100 tabular-nums">
                  {fmt(p.size)}
                </span>
              </span>
            </span>
          ))}
        </div>

        <dl className="mt-6 grid grid-cols-3 gap-3 border-t border-ink-700/40 pt-5">
          <Stat label="Total items" value={fmt(result.total_items)} />
          <Stat label="Total packs" value={fmt(result.total_packs)} />
          <Stat
            label="Overshoot"
            value={fmt(result.overshoot)}
            hint={order !== null && order > 0 ? `vs order of ${fmt(order)}` : undefined}
            accent={result.overshoot === 0 ? "ok" : "warn"}
          />
        </dl>
      </div>
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
      ? "text-emerald-300"
      : accent === "warn"
        ? "text-brand-200"
        : "text-ink-50";
  return (
    <div className="min-w-0">
      <dt className="font-mono text-[10px] uppercase tracking-[0.22em] text-ink-400">{label}</dt>
      <dd className={`mt-1.5 font-mono text-2xl font-semibold tabular-nums ${tone}`}>{value}</dd>
      {hint ? <dd className="mt-0.5 text-[11px] text-ink-500">{hint}</dd> : null}
    </div>
  );
}
