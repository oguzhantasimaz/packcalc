"use client";

import { useState } from "react";
import { toast } from "sonner";

import { calculate } from "@/lib/api";
import { ApiError, type CalculateResponse } from "@/lib/types";

import { ResultPanel } from "./ResultPanel";

type Parsed = { ok: true; value: number } | { ok: false; reason: string };

export function Calculator() {
  const [orderStr, setOrderStr] = useState("12001");
  const [result, setResult] = useState<CalculateResponse | null>(null);
  const [running, setRunning] = useState(false);

  const parsed: Parsed = (() => {
    const t = orderStr.trim();
    if (t === "") return { ok: false, reason: "Enter an order quantity." };
    const n = Number(t);
    if (!Number.isInteger(n) || n < 0) return { ok: false, reason: "Order must be a non-negative integer." };
    return { ok: true, value: n };
  })();

  const run = async () => {
    if (!parsed.ok) return;
    setRunning(true);
    try {
      const res = await calculate(parsed.value);
      setResult(res);
    } catch (e) {
      const err = e instanceof ApiError ? e : new ApiError(0, "unknown", String(e), "");
      toast.error(`Calculate failed: ${err.message}`, {
        description: err.requestId ? `request_id: ${err.requestId}` : undefined,
      });
    } finally {
      setRunning(false);
    }
  };

  return (
    <section className="glass animate-fade-up relative overflow-hidden rounded-2xl p-6 md:p-7">
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-brand-400/50 to-transparent" />

      <header className="flex items-baseline justify-between">
        <div>
          <div className="text-[10px] font-medium uppercase tracking-[0.22em] text-brand-300">
            02 · Calculate
          </div>
          <h2 className="mt-1 font-display text-2xl text-ink-50">Order</h2>
        </div>
        <span className="font-mono text-[11px] uppercase tracking-wider text-ink-400">
          POST · /calculate
        </span>
      </header>

      <p className="mt-2 max-w-md text-sm leading-relaxed text-ink-300">
        Enter a quantity. The API picks the combination that minimizes total items shipped, then the
        pack count.
      </p>

      <div className="mt-6">
        <label className="text-[10px] font-medium uppercase tracking-[0.22em] text-ink-400">
          Items ordered
        </label>
        <div className="mt-2 flex flex-col gap-3 sm:flex-row">
          <div className="relative flex-1">
            <input
              type="number"
              inputMode="numeric"
              min={0}
              step={1}
              value={orderStr}
              onChange={(e) => setOrderStr(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void run();
              }}
              className="peer w-full rounded-lg border border-ink-700/60 bg-ink-900/60 px-4 py-3.5 font-mono text-2xl tabular-nums text-ink-50 outline-none transition focus:border-brand-400/60 focus:bg-ink-900 focus:ring-2 focus:ring-brand-400/20"
              placeholder="0"
            />
            <span className="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 text-[10px] uppercase tracking-wider text-ink-500">
              units
            </span>
          </div>
          <button
            type="button"
            onClick={run}
            disabled={!parsed.ok || running}
            className="group inline-flex items-center justify-center rounded-lg bg-brand-400 px-5 py-3 text-sm font-semibold tracking-wide text-ink-950 shadow-glow-sm transition hover:bg-brand-300 hover:shadow-glow-md disabled:cursor-not-allowed disabled:bg-ink-700 disabled:text-ink-500 disabled:shadow-none sm:px-7"
          >
            {running ? (
              <span className="inline-flex items-center gap-2">
                <span className="h-3 w-3 animate-spin rounded-full border-2 border-ink-950/30 border-t-ink-950" />
                Calculating…
              </span>
            ) : (
              <span className="inline-flex items-center gap-2">
                Calculate
                <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                  <path d="M2 6 H10 M6 2 L10 6 L6 10" stroke="currentColor" strokeWidth="1.8" strokeLinecap="square" />
                </svg>
              </span>
            )}
          </button>
        </div>
        {!parsed.ok ? <p className="mt-2 text-xs text-red-300">{parsed.reason}</p> : null}
      </div>

      <div className="mt-6">
        <ResultPanel result={result} order={parsed.ok ? parsed.value : null} />
      </div>
    </section>
  );
}
