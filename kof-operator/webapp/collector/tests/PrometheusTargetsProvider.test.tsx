import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";
import usePrometheusTarget from "../src/providers/prometheus/PrometheusHook";
import { ClustersData } from "../src/models/Cluster";
import { fakeData } from "./fake_data/fake_response";

const TestComponent = () => {
  const { loading, error, data, filteredData, addFilter, clearFilters } =
    usePrometheusTarget();

  return (
    <div>
      <div data-testid="loading">{loading ? "LOADING" : "NOT_LOADING"}</div>
      <div data-testid="error">{error?.message || "NO_ERROR"}</div>
      <div data-testid="data">{data ? "HAS_DATA" : "NO_DATA"}</div>
      <div data-testid="filtered">
        {filteredData ? filteredData.length : "NO_FILTERED"}
      </div>
      <button
        onClick={() => addFilter("test", () => [])}
        data-testid="add-filter"
      >
        Add Filter
      </button>
      <button onClick={() => clearFilters()} data-testid="clear-filters">
        Clear Filters
      </button>
    </div>
  );
};

describe("Prometheus target provider", () => {
  const fakeClusters: ClustersData = fakeData;

  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(fakeClusters),
      })
    );
  });

  it("should fetch data and provide it", async () => {
    render(
      <PrometheusTargetProvider>
        <TestComponent />
      </PrometheusTargetProvider>
    );

    expect(screen.getByTestId("loading").textContent).toBe("LOADING");

    await waitFor(() =>
      expect(screen.getByTestId("loading").textContent).toBe("NOT_LOADING")
    );

    expect(screen.getByTestId("data").textContent).toBe("HAS_DATA");
    expect(screen.getByTestId("filtered").textContent).toBe("2");
    expect(screen.getByTestId("error").textContent).toBe("NO_ERROR");
  });

  it("should handle fetch error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => Promise.resolve({ ok: false, status: 500 }))
    );

    render(
      <PrometheusTargetProvider>
        <TestComponent />
      </PrometheusTargetProvider>
    );

    await waitFor(() =>
      expect(screen.getByTestId("loading").textContent).toBe("NOT_LOADING")
    );

    expect(screen.getByTestId("error").textContent).toMatch('HTTP error');
    expect(screen.getByTestId("data").textContent).toBe("NO_DATA");
  });

  it("should add and clear filters", async () => {
    render(
      <PrometheusTargetProvider>
        <TestComponent />
      </PrometheusTargetProvider>
    );

    await waitFor(() =>
      expect(screen.getByTestId("data").textContent).toBe("HAS_DATA")
    );

    screen.getByTestId("add-filter").click();
    await waitFor(() =>
      expect(screen.getByTestId("filtered").textContent).toBe("0")
    );

    screen.getByTestId("clear-filters").click();
    await waitFor(() =>
      expect(screen.getByTestId("filtered").textContent).toBe("2")
    );
  });
});
