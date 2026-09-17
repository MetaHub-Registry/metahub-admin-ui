import { screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import InventoryPage from "./InventoryPage";
import { renderWithProviders } from "@/test/test-utils";

afterEach(() => {
  vi.unstubAllGlobals();
});

const snapshot = {
  entries: [
    { ip: "10.10.1.42", profile: "gpu/a100", hostname: "gpu-h100-042.a100.cluster.example.com" },
    { cidr: "10.0.1.0/24", profile: "zen4" },
    { ip: "192.168.5.115", profile: "e2e/42", runtime: true },
  ],
  hostname: {
    lookup_ttl: "1m0s",
    rules: [{ pattern: `\\.a100\\.cluster\\.example\\.com$`, profile: "gpu/a100" }],
  },
};

describe("InventoryPage", () => {
  it("renders entries and hostname rules", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(JSON.stringify(snapshot), { status: 200 }))
    );
    renderWithProviders(<InventoryPage />);

    expect(await screen.findByText("10.10.1.42")).toBeInTheDocument();
    expect(screen.getByText("10.0.1.0/24")).toBeInTheDocument();
    // gpu/a100 appears twice (exact-IP entry + hostname rule)
    expect(screen.getAllByText("gpu/a100").length).toBe(2);
    expect(screen.getByText("zen4")).toBeInTheDocument();
    // runtime entry: flagged with a badge + the e2e profile visible
    expect(screen.getByText("runtime")).toBeInTheDocument();
    expect(screen.getByText("e2e/42")).toBeInTheDocument();
    // rule pattern is rendered verbatim (single backslashes)
    expect(screen.getByText("\\.a100\\.cluster\\.example\\.com$")).toBeInTheDocument();
    expect(screen.getByText(/lookup cache ttl/i)).toBeInTheDocument();
  });

  it("renders empty states", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({ entries: [], hostname: { lookup_ttl: "1m0s", rules: [] } }),
          { status: 200 }
        )
      )
    );
    renderWithProviders(<InventoryPage />);

    expect(await screen.findByText(/no exact-ip or cidr entries configured/i)).toBeInTheDocument();
    expect(screen.getByText(/no hostname rules configured/i)).toBeInTheDocument();
  });

  it("surfaces API errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("unauthorized", { status: 401 }))
    );
    renderWithProviders(<InventoryPage />);

    expect(await screen.findByText(/inventory failed \(401\)/i)).toBeInTheDocument();
  });
});
