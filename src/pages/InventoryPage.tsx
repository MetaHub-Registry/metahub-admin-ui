import { useQuery } from "@tanstack/react-query";
import { getInventory, type InventoryEntry } from "@/lib/api";

function entryKind(e: InventoryEntry): string {
  if (e.ip) return "exact-ip";
  if (e.cidr) return "cidr";
  return "?";
}

export default function InventoryPage() {
  const query = useQuery({
    queryKey: ["inventory"],
    queryFn: ({ signal }) => getInventory(signal),
  });

  if (query.isPending) {
    return <p className="text-sm text-slate-500">Loading inventory…</p>;
  }
  if (query.isError) {
    return (
      <div className="rounded-lg border border-red-300 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300">
        {query.error instanceof Error ? query.error.message : "failed to load inventory"}
      </div>
    );
  }

  const inv = query.data;

  return (
    <section className="w-full max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight">Inventory</h1>
      <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
        Configured sources — evaluated in order, first match wins.
      </p>

      <h2 className="mt-8 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Entries ({inv.entries.length})
      </h2>
      {inv.entries.length === 0 ? (
        <p className="mt-2 text-sm text-slate-400 dark:text-slate-500">
          No exact-IP or CIDR entries configured.
        </p>
      ) : (
        <ul className="mt-2 divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white dark:divide-slate-800 dark:border-slate-800 dark:bg-slate-900">
          {inv.entries.map((e, i) => (
            <li key={i} className="flex items-center gap-3 px-4 py-3">
              <span
                className={`rounded px-1.5 py-0.5 text-xs font-medium ${
                  e.ip
                    ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300"
                    : "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300"
                }`}
              >
                {entryKind(e)}
              </span>
              {e.runtime && (
                <span
                  className="rounded bg-violet-100 px-1.5 py-0.5 text-xs font-medium text-violet-800 dark:bg-violet-900/40 dark:text-violet-300"
                  title="Ephemeral runtime entry (POST /api/v1/entries) — wiped on restart"
                >
                  runtime
                </span>
              )}
              <span className="font-mono text-sm">{e.ip ?? e.cidr}</span>
              <span className="ml-auto font-mono text-sm text-sky-700 dark:text-sky-400">
                {e.profile}
              </span>
              {e.hostname && (
                <span className="hidden break-all text-xs text-slate-400 sm:block" title={e.hostname}>
                  {e.hostname}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}

      <h2 className="mt-8 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Hostname rules ({inv.hostname.rules.length})
      </h2>
      <p className="mt-1 text-xs text-slate-400 dark:text-slate-500">
        DNS PTR of the client IP, matched case-insensitively · lookup cache TTL:{" "}
        <span className="font-mono">{inv.hostname.lookup_ttl}</span>
      </p>
      {inv.hostname.rules.length === 0 ? (
        <p className="mt-2 text-sm text-slate-400 dark:text-slate-500">No hostname rules configured.</p>
      ) : (
        <ol className="mt-2 divide-y divide-slate-200 rounded-lg border border-slate-200 bg-white dark:divide-slate-800 dark:border-slate-800 dark:bg-slate-900">
          {inv.hostname.rules.map((r, i) => (
            <li key={i} className="flex items-center gap-3 px-4 py-3">
              <span className="font-mono text-xs text-slate-400">#{i}</span>
              <code className="break-all font-mono text-sm">{r.pattern}</code>
              <span className="ml-auto font-mono text-sm text-sky-700 dark:text-sky-400">
                {r.profile}
              </span>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
