import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import MainPage from "../src/components/features/MainPage";
import { ClustersData } from "../src/models/Cluster";
import { fakeData } from "./fake_data/fake_response";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";

window.HTMLElement.prototype.scrollIntoView = function () {};

describe("MainPage with all filters", () => {
  const fakeClusters: ClustersData = fakeData;
  const manager = new PrometheusTargetsManager(fakeClusters);

  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(fakeClusters),
      })
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

    const upCheckbox = screen.getByRole("up-checkbox") as HTMLInputElement;
    expect(upCheckbox).toBeInTheDocument();

    const unknownCheckbox = screen.getByRole(
      "unknown-checkbox"
    ) as HTMLInputElement;
    expect(unknownCheckbox).toBeInTheDocument();

    const downCheckbox = screen.getByRole("down-checkbox") as HTMLInputElement;
    expect(downCheckbox).toBeInTheDocument();

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
    const upCheckbox = screen.getByRole("up-checkbox") as HTMLInputElement;
    const unknownCheckbox = screen.getByRole(
      "unknown-checkbox"
    ) as HTMLInputElement;
    const downCheckbox = screen.getByRole("down-checkbox") as HTMLInputElement;
    const clusterPopoverBtn = popoverBtn[0];
    await waitFor(() => {
      expect(clusterPopoverBtn).not.toBeDisabled();
      expect(input).not.toBeDisabled();
      expect(upCheckbox).not.toBeDisabled();
      expect(unknownCheckbox).not.toBeDisabled();
      expect(downCheckbox).not.toBeDisabled();
    });

    const clusterName = manager.clusters[0].name;
    await userEvent.click(clusterPopoverBtn);
    await userEvent.click(await screen.findAllByRole("option")[0]);
    
    await userEvent.click(upCheckbox);

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
