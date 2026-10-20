import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import LookupPage from "./LookupPage";
import { renderWithProviders } from "@/test/test-utils";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("LookupPage", () => {
  it("renders the resolve form", () => {
    renderWithProviders(<LookupPage />);
    expect(screen.getByRole("heading", { name: /ip lookup/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /resolve/i })).toBeInTheDocument();
  });

  it("shows the resolve result incl. source", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ip: "10.10.1.42", profile: "gpu/a100", source: "ip:0" }), {
          status: 200,
        })
      )
    );
    renderWithProviders(<LookupPage />);
    await userEvent.type(screen.getByLabelText(/ip address/i), "10.10.1.42");
    await userEvent.click(screen.getByRole("button", { name: /resolve/i }));

    expect(await screen.findByText("gpu/a100")).toBeInTheDocument();
    expect(screen.getByText("ip:0")).toBeInTheDocument();
  });

  it("renders the no-match state for a 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("{}", { status: 404 }))
    );
    renderWithProviders(<LookupPage />);
    await userEvent.type(screen.getByLabelText(/ip address/i), "172.16.9.9");
    await userEvent.click(screen.getByRole("button", { name: /resolve/i }));

    expect(await screen.findByText(/no match for/i)).toBeInTheDocument();
  });

  it("surfaces API errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("invalid ip", { status: 400 }))
    );
    renderWithProviders(<LookupPage />);
    await userEvent.type(screen.getByLabelText(/ip address/i), "not-an-ip");
    await userEvent.click(screen.getByRole("button", { name: /resolve/i }));

    expect(await screen.findByText(/resolve failed \(400\)/i)).toBeInTheDocument();
  });
});
