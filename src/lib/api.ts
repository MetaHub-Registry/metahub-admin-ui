/**
 * Typed API client for the metahub-admin-ui surface (/api/v1).
 */

export interface ResolveResult {
  ip: string;
  profile: string;
  hostname?: string;
  source: string;
}

export interface InventoryEntry {
  ip?: string;
  cidr?: string;
  profile: string;
  hostname?: string;
  /** Ephemeral overlay entry (POST /api/v1/entries) — wiped on restart. */
  runtime?: boolean;
}

export interface InventorySnapshot {
  entries: InventoryEntry[];
  hostname: {
    lookup_ttl: string;
    rules: { pattern: string; profile: string }[];
  };
}

export async function resolveIP(ip: string, signal?: AbortSignal): Promise<ResolveResult | null> {
  const res = await fetch(`/api/v1/resolve?ip=${encodeURIComponent(ip)}`, { signal });
  if (res.status === 404) {
    return null; // documented no-match — not an error for the UI
  }
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`resolve failed (${res.status}): ${body}`);
  }
  return (await res.json()) as ResolveResult;
}

export async function getInventory(signal?: AbortSignal): Promise<InventorySnapshot> {
  const res = await fetch("/api/v1/inventory", { signal });
  if (!res.ok) {
    throw new Error(`inventory failed (${res.status})`);
  }
  return (await res.json()) as InventorySnapshot;
}
