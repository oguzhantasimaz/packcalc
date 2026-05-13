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

function parseRows(rows: Row[]): { ok: true; sizes: number[] } | { ok: false; reason: string } {
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
  const [etag, setEtag] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [nextId, setNextId] = useState(1);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const { data, etag } = await getPacks();
      setRows(data.sizes.map((s, i) => makeRow(i + 1, s)));
      setNextId(data.sizes.length + 1);
      setEtag(etag);
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

  const removeRow = (id: number) => {
    setRows((rs) => rs.filter((r) => r.id !== id));
  };

  const updateRow = (id: number, value: string) => {
    setRows((rs) => rs.map((r) => (r.id === id ? { ...r, value } : r)));
  };

  const save = async () => {
    if (!validation.ok) return;
    setSaving(true);
    try {
      const { data, etag: newEtag } = await putPacks(validation.sizes, etag);
      setRows(data.sizes.map((s, i) => makeRow(i + 1, s)));
      setNextId(data.sizes.length + 1);
      setEtag(newEtag);
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
    <section className="rounded-2xl border border-zinc-200 bg-white/80 p-6 shadow-sm backdrop-blur dark:border-zinc-800 dark:bg-zinc-900/60">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Pack sizes</h2>
        <span className="font-mono text-xs text-zinc-400 dark:text-zinc-500">
          {etag ? `v ${etag.slice(0, 8)}…` : "—"}
        </span>
      </div>
      <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
        Whole-pack quantities the warehouse can ship. Changes apply immediately on save.
      </p>

      <div className="mt-5 space-y-2">
        {loading && rows.length === 0 ? (
          <div className="rounded-lg bg-zinc-100 px-3 py-6 text-center text-sm text-zinc-500 dark:bg-zinc-800/50">
            Loading current pack sizes…
          </div>
        ) : (
          rows.map((r) => (
            <div key={r.id} className="flex items-center gap-2">
              <input
                type="number"
                inputMode="numeric"
                min={1}
                step={1}
                value={r.value}
                onChange={(e) => updateRow(r.id, e.target.value)}
                className="w-full rounded-lg border border-zinc-200 bg-white px-3 py-2 font-mono text-sm tabular-nums shadow-sm outline-none transition focus:border-amber-500 focus:ring-1 focus:ring-amber-500 dark:border-zinc-700 dark:bg-zinc-950 dark:focus:border-amber-400 dark:focus:ring-amber-400"
                placeholder="e.g. 500"
              />
              <button
                type="button"
                onClick={() => removeRow(r.id)}
                aria-label="Remove pack size"
                className="rounded-lg border border-zinc-200 px-2.5 py-2 text-sm text-zinc-500 transition hover:border-red-400 hover:text-red-500 dark:border-zinc-700"
              >
                ×
              </button>
            </div>
          ))
        )}
      </div>

      <div className="mt-3 flex items-center justify-between">
        <button
          type="button"
          onClick={addRow}
          className="text-sm font-medium text-amber-600 transition hover:text-amber-700 dark:text-amber-400 dark:hover:text-amber-300"
        >
          + Add size
        </button>
        {!validation.ok && rows.length > 0 ? (
          <span className="text-xs text-red-500">{validation.reason}</span>
        ) : (
          <span className="text-xs text-zinc-400 dark:text-zinc-500">
            {rows.length} row{rows.length === 1 ? "" : "s"}
          </span>
        )}
      </div>

      <div className="mt-6 flex justify-end">
        <button
          type="button"
          onClick={save}
          disabled={!validation.ok || saving || loading}
          className="inline-flex items-center justify-center rounded-lg bg-amber-500 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-amber-600 disabled:cursor-not-allowed disabled:bg-zinc-200 disabled:text-zinc-400 dark:disabled:bg-zinc-800 dark:disabled:text-zinc-600"
        >
          {saving ? "Saving…" : "Save pack sizes"}
        </button>
      </div>
    </section>
  );
}
