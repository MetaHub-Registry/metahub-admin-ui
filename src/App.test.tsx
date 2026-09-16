import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import App from "./App";
import { renderWithProviders } from "./test/test-utils";

describe("App", () => {
  it("renders the nav with both pages", () => {
    renderWithProviders(<App />);
    expect(screen.getByRole("link", { name: "MetaHub Admin" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Lookup" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Inventory" })).toBeInTheDocument();
    // Default route is the lookup page.
    expect(screen.getByRole("heading", { name: /ip lookup/i })).toBeInTheDocument();
  });
});
