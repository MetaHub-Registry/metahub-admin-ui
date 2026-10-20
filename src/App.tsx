import { Link, Route, Routes } from "react-router-dom";
import InventoryPage from "@/pages/InventoryPage";
import LookupPage from "@/pages/LookupPage";

export default function App() {
  return (
    <div className="flex min-h-svh flex-col bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <header className="border-b border-slate-200 dark:border-slate-800">
        <div className="mx-auto flex w-full max-w-3xl items-center gap-6 px-6 py-4">
          <Link to="/" className="text-base font-semibold tracking-tight">
            MetaHub Admin
          </Link>
          <nav className="flex gap-4 text-sm">
            <Link
              to="/"
              className="text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Lookup
            </Link>
            <Link
              to="/inventory"
              className="text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Inventory
            </Link>
          </nav>
        </div>
      </header>
      <main className="mx-auto w-full max-w-3xl flex-1 px-6 py-10">
        <Routes>
          <Route path="/" element={<LookupPage />} />
          <Route path="/inventory" element={<InventoryPage />} />
          <Route path="*" element={<LookupPage />} />
        </Routes>
      </main>
      <footer className="px-6 pb-6 text-center text-xs text-slate-400 dark:text-slate-600">
        ops-facing admin service · inventory lookup ·
        <span className="font-mono"> /api/v1</span>
      </footer>
    </div>
  );
}
