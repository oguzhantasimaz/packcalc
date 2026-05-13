"use client";

import { useState } from "react";
import { toast } from "sonner";

import { calculate } from "@/lib/api";
import { ApiError, type CalculateResponse } from "@/lib/types";

import { ResultPanel } from "./ResultPanel";

export function Calculator() {
  const [orderStr, setOrderStr] = useState("12001");
  const [result, setResult] = useState<CalculateResponse | null>(null);
  const [running, setRunning] = useState(false);

  type Parsed = { ok: true; value: number } | { ok: false; reason: string };
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
    <section className="rounded-2xl border border-zinc-200 bg-white/80 p-6 shadow-sm backdrop-blur dark:border-zinc-800 dark:bg-zinc-900/60">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Calculate an order</h2>
        <span className="text-xs text-zinc-400 dark:text-zinc-500">POST /calculate</span>
      </div>
      <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
        Enter an order quantity. The API picks the combination minimizing total items, then packs.
      </p>

      <div className="mt-5 flex flex-col gap-3 sm:flex-row">
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
          className="flex-1 rounded-lg border border-zinc-200 bg-white px-3 py-2 font-mono text-base tabular-nums shadow-sm outline-none transition focus:border-amber-500 focus:ring-1 focus:ring-amber-500 dark:border-zinc-700 dark:bg-zinc-950 dark:focus:border-amber-400 dark:focus:ring-amber-400"
          placeholder="e.g. 12001"
        />
        <button
          type="button"
          onClick={run}
          disabled={!parsed.ok || running}
          className="inline-flex items-center justify-center rounded-lg bg-amber-500 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-amber-600 disabled:cursor-not-allowed disabled:bg-zinc-200 disabled:text-zinc-400 dark:disabled:bg-zinc-800 dark:disabled:text-zinc-600"
        >
          {running ? "Calculating…" : "Calculate"}
        </button>
      </div>

      {!parsed.ok ? <p className="mt-2 text-xs text-red-500">{parsed.reason}</p> : null}

      <div className="mt-6">
        <ResultPanel result={result} order={parsed.ok ? parsed.value : null} />
      </div>
    </section>
  );
}
