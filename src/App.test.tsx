import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import App from "./App";
import { renderWithProviders } from "./test/test-utils";

describe("App", () => {
  it("renders the admin headline", () => {
    renderWithProviders(<App />);
    expect(
      screen.getByRole("heading", { name: /metahub admin/i })
    ).toBeInTheDocument();
  });

  it("points at the openspec change for the upcoming pages", () => {
    renderWithProviders(<App />);
    expect(screen.getByText(/add-inventory-service/)).toBeInTheDocument();
  });
});
