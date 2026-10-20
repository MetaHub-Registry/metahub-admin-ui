import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { resolveIP, type ResolveResult } from "@/lib/api";

/** Source badge coloring per tier. */
function sourceBadgeClass(source: string): string {
  if (source.startsWith("ip:")) {
    return "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300";
  }
  if (source.startsWith("hostname:")) {
    return "bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-300";
  }
  return "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300";
}

export default function LookupPage() {
  const [ip, setIP] = useState("");
  const [lookup, setLookup] = useState<string | null>(null);

  const query = useMutation({
    mutationFn: (ip: string) => resolveIP(ip),
    mutationKey: ["resolve"],
  });

  const result: ResolveResult | null | undefined = query.data;
  const error = query.error instanceof Error ? query.error.message : null;

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = ip.trim();
    if (!trimmed) return;
    setLookup(trimmed);
    query.mutate(trimmed);
  }

  return (
    <section className="w-full max-w-xl">
      <h1 className="text-2xl font-semibold tracking-tight">IP Lookup</h1>
      <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
        Which profile does this client IP resolve to?
      </p>

      <form onSubmit={onSubmit} className="mt-6 flex gap-2">
        <input
          value={ip}
          onChange={(e) => setIP(e.target.value)}
          placeholder="10.10.1.42"
          aria-label="IP address"
          className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 font-mono text-sm outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-900 dark:focus:ring-sky-900"
        />
        <button
          type="submit"
          disabled={query.isPending}
          className="shrink-0 rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-white"
        >
          {query.isPending ? "Resolving…" : "Resolve"}
        </button>
      </form>

      <div className="mt-6" aria-live="polite">
        {query.isPending && (
          <p className="text-sm text-slate-500">Resolving {lookup}…</p>
        )}

        {query.isSuccess && result !== null && result !== undefined && (
          <div className="rounded-lg border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900">
            <div className="flex items-center justify-between gap-3">
              <span className="font-mono text-sm text-slate-500 dark:text-slate-400">{result.ip}</span>
              <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${sourceBadgeClass(result.source)}`}>
                {result.source}
              </span>
            </div>
            <p className="mt-3 font-mono text-2xl font-semibold">{result.profile}</p>
            {result.hostname && (
              <p className="mt-1 break-all text-sm text-slate-500 dark:text-slate-400">{result.hostname}</p>
            )}
          </div>
        )}

        {query.isSuccess && result === null && (
          <div className="rounded-lg border border-dashed border-slate-300 p-5 text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
            No match for <span className="font-mono">{lookup}</span> — no exact-IP entry, no
            hostname rule, no CIDR entry applies.
          </div>
        )}

        {error && (
          <div className="rounded-lg border border-red-300 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300">
            {error}
          </div>
        )}
      </div>
    </section>
  );
}
