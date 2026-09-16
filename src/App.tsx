/**
 * Skeleton page. The real surfaces — IP lookup and the entries/rules
 * overview — land with openspec change `add-inventory-service`, task 2.4.
 */
export default function App() {
  return (
    <div className="flex min-h-svh flex-col items-center justify-center bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <main className="mx-auto max-w-xl px-6 text-center">
        <h1 className="text-3xl font-semibold tracking-tight">MetaHub Admin</h1>
        <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">
          Ops-facing admin service — skeleton.
        </p>
        <p className="mt-8 rounded-md border border-dashed border-slate-300 p-6 text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
          Inventory lookup and the rules overview are implemented in{" "}
          <code className="rounded bg-slate-100 px-1.5 py-0.5 dark:bg-slate-800">
            openspec/changes/add-inventory-service
          </code>{" "}
          (task 2.4).
        </p>
      </main>
    </div>
  );
}
