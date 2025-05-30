import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import SearchBar from "../src/components/features/SearchBar";
import { ClustersData } from "../src/models/Cluster";
import { fakeData } from "./fake_data/fake_response";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";
import TargetList from "../src/components/features/TargetsList";

describe("SearchBar Component", () => {
  const fakeClusters: ClustersData = fakeData;
  const manager = new PrometheusTargetsManager(fakeClusters);

  beforeEach(() => {
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

  it("should render search input element", () => {
    render(
      <PrometheusTargetProvider>
        <SearchBar />
      </PrometheusTargetProvider>
    );

    const input = screen.getByRole("textbox");
    expect(input).toBeTruthy();
  });

  it("should disable input when loading", async () => {
    render(
      <PrometheusTargetProvider>
        <SearchBar />
      </PrometheusTargetProvider>
    );

    const input = screen.getByRole("textbox") as HTMLInputElement;
    expect(input.disabled).toBe(true);

    await waitFor(() => expect(input.disabled).toBe(false));
  });

  it("should filter the list based on input", async () => {
    render(
      <PrometheusTargetProvider>
        <SearchBar />
        <TargetList />
      </PrometheusTargetProvider>
    );

    const input = screen.getByRole("textbox") as HTMLInputElement;
    await waitFor(() => expect(input.disabled).toBe(false));

    const allRows = screen.getAllByRole("row");

    expect(allRows).toHaveLength(
      manager.targets.length + manager.clusters.length
    );

    const firstTarget: string = manager.clusters[0].nodes[0].targets[0].scrapeUrl;

    await userEvent.type(input, firstTarget);
    const filteredRows = screen.getAllByRole("row");

    // 1 header + 1 target
    expect(filteredRows).toHaveLength(2);
    expect(screen.getByText(firstTarget)).toBeInTheDocument();
  });

  it("should restore the full list when input is cleared", async () => {
    render(
      <PrometheusTargetProvider>
        <SearchBar />
        <TargetList />
      </PrometheusTargetProvider>
    );

    const input = screen.getByRole("textbox") as HTMLInputElement;
    await waitFor(() => expect(input.disabled).toBe(false));

    await userEvent.type(input, "hello-world");
    await waitFor(() => {
      const rowsAfterFilter = screen.queryAllByRole("row");
      expect(rowsAfterFilter.length).toBe(0);
    });

    await userEvent.clear(input);
    await waitFor(() => {
      const rows = screen.getAllByRole("row");
      expect(rows).toHaveLength(
        manager.targets.length + manager.clusters.length
      );
    });
  });
});
