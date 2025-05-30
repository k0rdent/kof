import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import MainPage from "../src/components/features/MainPage";
import { ClustersData } from "../src/models/Cluster";
import { fakeData } from "./fake_data/fake_response";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";

global.ResizeObserver = require("resize-observer-polyfill");
window.HTMLElement.prototype.scrollIntoView = function () {};

describe("MainPage with all filters", () => {
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

  it("renders cluster popover, search bar, health selector, and target list", async () => {
    render(
      <PrometheusTargetProvider>
        <MainPage />
      </PrometheusTargetProvider>
    );

    const searchInput = await screen.findByRole("textbox");
    expect(searchInput).toBeInTheDocument();

    const popovers = screen.getAllByRole("combobox");
    popovers.forEach((popover) => {
      expect(popover).toBeInTheDocument();
    });

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    expect(screen.getByText("0 Unknown")).toBeInTheDocument();
    expect(screen.getByText("0 Down")).toBeInTheDocument();
    expect(screen.getByText("0 Up")).toBeInTheDocument();

    const rows = screen.getAllByRole("row");
    expect(rows).toHaveLength(manager.targets.length + manager.clusters.length);
  });

  it("combines cluster, search, and health filters", async () => {
    render(
      <PrometheusTargetProvider>
        <MainPage />
      </PrometheusTargetProvider>
    );

    // wait initial load
    const popoverBtn = await screen.findAllByRole("combobox");
    const input = screen.getByRole("textbox");
    const checkboxes = await screen.findAllByRole("checkbox");
    const clusterPopoverBtn = popoverBtn[0];
    await waitFor(() => {
      expect(clusterPopoverBtn).not.toBeDisabled();
      expect(input).not.toBeDisabled();
      checkboxes.forEach((cb) => expect(cb).not.toBeDisabled());
    });

    const clusterName = manager.clusters[0].name;
    await userEvent.click(clusterPopoverBtn);
    await userEvent.click(await screen.findAllByRole("option")[0]);

    const downCheckbox = checkboxes.find((cb) =>
      cb.nextSibling?.textContent?.includes("Up")
    )!;
    await userEvent.click(downCheckbox);

    const expectedTargets = manager
      .findCluster(clusterName)!
      .targets.filter((t) => t.health === "up");

    const term = expectedTargets[0].scrapeUrl;
    await userEvent.type(input, term);

    const rows = screen.getAllByRole("row");

    // header + one match
    expect(rows).toHaveLength(2);
    expect(screen.getByText(term)).toBeInTheDocument();
  });
});
