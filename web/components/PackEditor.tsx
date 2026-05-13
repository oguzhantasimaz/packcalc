"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import { getPacks, putPacks } from "@/lib/api";
import { ApiError } from "@/lib/types";

interface Row {
  id: number;
  value: string;
}

function makeRow(id: number, value: number | string = ""): Row {
  return { id, value: typeof value === "number" ? String(value) : value };
}

type Parsed = { ok: true; sizes: number[] } | { ok: false; reason: string };

function parseRows(rows: Row[]): Parsed {
  if (rows.length === 0) return { ok: false, reason: "At least one pack size is required." };
  const sizes: number[] = [];
  const seen = new Set<number>();
  for (const r of rows) {
    const trimmed = r.value.trim();
    if (trimmed === "") return { ok: false, reason: "Each row must contain a number." };
    const n = Number(trimmed);
    if (!Number.isInteger(n) || n <= 0) return { ok: false, reason: `"${trimmed}" is not a positive integer.` };
    if (seen.has(n)) return { ok: false, reason: `${n} is duplicated.` };
    seen.add(n);
    sizes.push(n);
  }
  return { ok: true, sizes };
}

export function PackEditor() {
  const [rows, setRows] = useState<Row[]>([]);
  const [savedSizes, setSavedSizes] = useState<number[]>([]);
  const [etag, setEtag] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [nextId, setNextId] = useState(1);

  const applyFetched = (sizes: number[], newEtag: string) => {
    setRows(sizes.map((s, i) => makeRow(i + 1, s)));
    setNextId(sizes.length + 1);
    setSavedSizes(sizes);
    setEtag(newEtag);
  };

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const { data, etag } = await getPacks();
      applyFetched(data.sizes, etag);
    } catch (e) {
      const err = e instanceof ApiError ? e : new ApiError(0, "unknown", String(e), "");
      toast.error(`Failed to load packs: ${err.message}`, {
        description: err.requestId ? `request_id: ${err.requestId}` : undefined,
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const validation = useMemo(() => parseRows(rows), [rows]);

  const addRow = () => {
    setRows((rs) => [...rs, makeRow(nextId)]);
    setNextId((n) => n + 1);
  };
  const removeRow = (id: number) => setRows((rs) => rs.filter((r) => r.id !== id));
  const updateRow = (id: number, value: string) =>
    setRows((rs) => rs.map((r) => (r.id === id ? { ...r, value } : r)));

  // isDirty = current rows differ from the last successfully fetched/saved
  // snapshot. Drives the Reset button and the Save button's enabled state.
  const isDirty = useMemo(() => {
    if (rows.length !== savedSizes.length) return true;
    for (let i = 0; i < rows.length; i++) {
      if (rows[i].value.trim() !== String(savedSizes[i])) return true;
    }
    return false;
  }, [rows, savedSizes]);

  const save = async () => {
    if (!validation.ok) return;
    setSaving(true);
    try {
      const { data, etag: newEtag } = await putPacks(validation.sizes, etag);
      applyFetched(data.sizes, newEtag);
      toast.success(`Saved ${data.sizes.length} pack size${data.sizes.length === 1 ? "" : "s"}.`);
    } catch (e) {
      const err = e instanceof ApiError ? e : new ApiError(0, "unknown", String(e), "");
      if (err.code === "version_mismatch") {
        toast.warning("Pack sizes were changed elsewhere. Refreshing…", {
          description: err.requestId ? `request_id: ${err.requestId}` : undefined,
        });
        await refresh();
      } else {
        toast.error(`Save failed: ${err.message}`, {
          description: err.requestId ? `request_id: ${err.requestId}` : undefined,
        });
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="glass animate-fade-up relative overflow-hidden rounded-2xl p-6 md:p-7">
      {/* hairline accent line at the top */}
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-brand-400/50 to-transparent" />

      <header className="flex items-baseline justify-between">
        <div>
          <div className="text-[10px] font-medium uppercase tracking-[0.22em] text-brand-300">
            01 · Configuration
          </div>
          <h2 className="mt-1 font-display text-2xl text-ink-50">Pack sizes</h2>
        </div>
        <span className="font-mono text-[11px] text-ink-400" title={etag}>
          {etag ? `v ${etag.slice(0, 8)}…` : "—"}
        </span>
      </header>

      <p className="mt-2 max-w-md text-sm leading-relaxed text-ink-300">
        Whole-pack quantities the warehouse can ship. Changes apply atomically on save with an
        <span className="mx-1 rounded bg-ink-800 px-1 py-0.5 font-mono text-[11px] text-brand-200">
          If-Match
        </span>
        precondition.
      </p>

      <div className="mt-6 space-y-2.5">
        {loading && rows.length === 0 ? (
          <div className="rounded-lg border border-ink-700/40 bg-ink-900/40 px-4 py-8 text-center text-sm text-ink-400">
            Loading current pack sizes…
          </div>
        ) : (
          rows.map((r, idx) => (
            <div key={r.id} className="group flex items-center gap-2">
              <span className="w-6 select-none text-right font-mono text-[11px] text-ink-500">
                {String(idx + 1).padStart(2, "0")}
              </span>
              <div className="relative flex-1">
                <input
                  type="number"
                  inputMode="numeric"
                  min={1}
                  step={1}
                  value={r.value}
                  onChange={(e) => updateRow(r.id, e.target.value)}
                  className="w-full rounded-lg border border-ink-700/60 bg-ink-900/60 px-3 py-2.5 font-mono text-sm tabular-nums text-ink-100 placeholder:text-ink-500 outline-none transition focus:border-brand-400/60 focus:bg-ink-900 focus:ring-2 focus:ring-brand-400/20"
                  placeholder="e.g. 500"
                />
                <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-[10px] uppercase tracking-wider text-ink-500">
                  items
                </span>
              </div>
              <button
                type="button"
                onClick={() => removeRow(r.id)}
                aria-label="Remove pack size"
                className="grid h-10 w-10 place-items-center rounded-lg border border-ink-700/60 text-ink-400 transition hover:border-red-400/40 hover:bg-red-500/10 hover:text-red-300"
              >
                <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                  <path d="M2 2 L10 10 M10 2 L2 10" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
                </svg>
              </button>
            </div>
          ))
        )}
      </div>

      <div className="mt-4 flex items-center justify-between">
        <button
          type="button"
          onClick={addRow}
          className="inline-flex items-center gap-1.5 text-xs font-medium uppercase tracking-[0.14em] text-brand-300 transition hover:text-brand-200"
        >
          <span className="grid h-4 w-4 place-items-center rounded-sm border border-brand-400/40 text-brand-300">
            +
          </span>
          Add size
        </button>
        {!validation.ok && rows.length > 0 ? (
          <span className="text-[11px] text-red-300">{validation.reason}</span>
        ) : (
          <span className="font-mono text-[11px] text-ink-500">
            {rows.length} row{rows.length === 1 ? "" : "s"}
          </span>
        )}
      </div>

      <div className="mt-7 flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={refresh}
          disabled={!isDirty || saving || loading || savedSizes.length === 0}
          title="Discard local changes and re-fetch the server's pack-size set"
          className="inline-flex items-center gap-1.5 rounded-lg border border-ink-700/60 px-3.5 py-2.5 text-xs font-medium uppercase tracking-[0.14em] text-ink-300 transition hover:border-brand-400/40 hover:text-brand-200 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-ink-700/60 disabled:hover:text-ink-300"
        >
          <svg width="11" height="11" viewBox="0 0 12 12" fill="none">
            <path
              d="M2 6 a4 4 0 1 1 1.2 2.83 M2 9 V6 H5"
              stroke="currentColor"
              strokeWidth="1.4"
              strokeLinecap="square"
              fill="none"
            />
          </svg>
          Reset
        </button>
        <button
          type="button"
          onClick={save}
          disabled={!validation.ok || !isDirty || saving || loading}
          className="group relative inline-flex items-center justify-center overflow-hidden rounded-lg bg-brand-400 px-5 py-2.5 text-sm font-semibold tracking-wide text-ink-950 shadow-glow-sm transition hover:bg-brand-300 hover:shadow-glow-md disabled:cursor-not-allowed disabled:bg-ink-700 disabled:text-ink-500 disabled:shadow-none"
        >
          <span className="relative z-10 inline-flex items-center gap-2">
            {saving ? (
              <>
                <span className="h-3 w-3 animate-spin rounded-full border-2 border-ink-950/30 border-t-ink-950" />
                Saving…
              </>
            ) : (
              <>
                Save pack sizes
                <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                  <path d="M2 5 H8 M5 2 L8 5 L5 8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="square" />
                </svg>
              </>
            )}
          </span>
        </button>
      </div>
    </section>
  );
}
