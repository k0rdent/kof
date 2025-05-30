import { describe, it, afterEach, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import TargetList from "../src/components/features/TargetsList";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import { fakeData } from "./fake_data/fake_response";
import { ClustersData } from "../src/models/Cluster";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";

describe("Target list", () => {
  const fakeClusters: ClustersData = fakeData;
  const manager = new PrometheusTargetsManager(fakeClusters);

  beforeEach(() => {
    // stub fetch to return successful response by default
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve({ ok: true, json: () => Promise.resolve(fakeClusters) })
      )
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should show spinner on initial load and then renders rows", async () => {
    const { container } = render(
      <PrometheusTargetProvider>
        <TargetList />
      </PrometheusTargetProvider>
    );

    expect(container.querySelector("svg.animate-spin")).toBeInTheDocument();

    await waitFor(() =>
      expect(container.querySelector("svg.animate-spin")).toBeNull()
    );
    const rows = screen.getAllByRole("row");

    // all targets + table header for each cluster
    expect(rows).toHaveLength(manager.targets.length + manager.clusters.length);
    manager.targets.forEach((t) => {
      expect(screen.getByText(t.scrapeUrl)).toBeInTheDocument();
    });
  });

  it("should render error state and allows reload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => Promise.resolve({ ok: false, status: 500 }))
    );

    const { container } = render(
      <PrometheusTargetProvider>
        <TargetList />
      </PrometheusTargetProvider>
    );

    // wait for error UI
    await waitFor(() =>
      expect(
        screen.getByText(
          `Failed to fetch prometheus targets. Click "Reload" button to try again.`
        )
      ).toBeInTheDocument()
    );

    const reload = screen.getByRole("button", { name: "Reload" });
    expect(reload).toBeEnabled();

    // stub successful fetch on retry
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve({ ok: true, json: () => Promise.resolve(fakeClusters) })
      )
    );

    fireEvent.click(reload);

    expect(container.querySelector("svg.animate-spin")).toBeInTheDocument();

    await waitFor(() =>
      expect(container.querySelector("svg.animate-spin")).toBeNull()
    );

    const rows = screen.getAllByRole("row");
    expect(rows).toHaveLength(manager.targets.length + manager.clusters.length);
  });

  it("should toggle JSON panel when clicking a row", async () => {
    const { container } = render(
      <PrometheusTargetProvider>
        <TargetList />
      </PrometheusTargetProvider>
    );

    expect(container.querySelector("svg.animate-spin")).toBeInTheDocument();

    await waitFor(() =>
      expect(container.querySelector("svg.animate-spin")).toBeNull()
    );
    const rows = screen.getAllByRole("row");
    const initialCount = rows.length;

    // click first data row
    fireEvent.click(rows[1]);
    await waitFor(() => {
      const newRows = screen.getAllByRole("row");
      expect(newRows.length).toBe(initialCount + 1);
    });
  });
});
