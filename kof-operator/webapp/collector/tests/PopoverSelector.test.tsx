import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import PopoverSelector, {
  PopoverClusterFilter,
} from "../src/components/features/PopoverSelector";
import { ClustersData } from "../src/models/Cluster";
import { fakeData } from "./fake_data/fake_response";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";
import TargetList from "../src/components/features/TargetsList";

global.ResizeObserver = require("resize-observer-polyfill");
window.HTMLElement.prototype.scrollIntoView = function () {};

describe("Popover selector", () => {
  const fakeClusters: ClustersData = fakeData;
  const manager = new PrometheusTargetsManager(fakeClusters);
  const mockOnSelectionChange = vi.fn();

  const defaultProps = {
    id: "test-popover",
    labelContent: "Select Clusters",
    noValuesText: "No clusters found",
    placeholderText: "Search clusters...",
    popoverButtonText: "Select clusters",
    dataToDisplay: manager.clusters,
    filterFn: PopoverClusterFilter,
    onSelectionChange: mockOnSelectionChange,
  };

  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve({ ok: true, json: () => Promise.resolve(fakeClusters) })
      )
    );
    mockOnSelectionChange.mockClear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should render popover selector", () => {
    render(
      <PrometheusTargetProvider>
        <PopoverSelector {...defaultProps} />
      </PrometheusTargetProvider>
    );

    expect(screen.getByText("Select Clusters")).toBeTruthy();
    expect(screen.getByRole("combobox")).toBeTruthy();
    expect(screen.getByText("Select clusters")).toBeTruthy();
  });

  it("should disable popover when loading", async () => {
    render(
      <PrometheusTargetProvider>
        <PopoverSelector {...defaultProps} />
      </PrometheusTargetProvider>
    );

    const button = screen.getByRole("combobox") as HTMLButtonElement;
    expect(button.disabled).toBe(true);

    await waitFor(() => expect(button).not.toBeDisabled());
  });

  it("should open popover when it is clicked", async () => {
    render(
      <PrometheusTargetProvider>
        <PopoverSelector {...defaultProps} />
      </PrometheusTargetProvider>
    );

    const button = screen.getByRole("combobox") as HTMLButtonElement;
    await waitFor(() => expect(button).not.toBeDisabled());

    await userEvent.click(button);

    const options = await screen.findAllByRole("option");
    expect(options).toHaveLength(2);

    expect(screen.getByText("aws-ue2-test-1")).toBeInTheDocument();
    expect(screen.getByText("aws-ue2-test-2")).toBeInTheDocument();
  });

  it("should filter items based on search input", async () => {
    render(
      <PrometheusTargetProvider>
        <PopoverSelector {...defaultProps} />
      </PrometheusTargetProvider>
    );

    const button = screen.getByRole("combobox");
    await waitFor(() => expect(button).not.toBeDisabled());

    await userEvent.click(button);

    const searchInput = screen.getByPlaceholderText("Search clusters...");
    const firstClusterName = manager.clusters[0].name;

    await userEvent.type(searchInput, firstClusterName);

    expect(screen.getByText(firstClusterName)).toBeInTheDocument();

    expect(
      screen.queryByText(manager.clusters[1].name)
    ).not.toBeInTheDocument();
  });

  it("should show no values text when search has no results", async () => {
    render(
      <PrometheusTargetProvider>
        <PopoverSelector {...defaultProps} />
      </PrometheusTargetProvider>
    );

    const button = screen.getByRole("combobox");
    await waitFor(() => expect(button).not.toBeDisabled());

    await userEvent.click(button);

    const searchInput = screen.getByPlaceholderText("Search clusters...");
    await userEvent.type(searchInput, "hello-world-cluster");

    expect(screen.getByText("No clusters found")).toBeInTheDocument();
  });

  it("should filter clusters when clicked", async () => {
    render(
      <PrometheusTargetProvider>
        <PopoverSelector {...defaultProps} />
        <TargetList />
      </PrometheusTargetProvider>
    );

    const button = screen.getByRole("combobox") as HTMLButtonElement;

    // waiting to load the ui
    await waitFor(() => {
      expect(button).not.toBeDisabled();

      const rows = screen.getAllByRole("row");
      expect(rows).toHaveLength(6);
    });

    // open combobox
    await userEvent.click(button);

    // get options frm the combobox
    const options = await screen.findAllByRole("option");
    expect(options).toHaveLength(2);

    // get first cluster name and option
    const firstClusterName = manager.clusters[0].name;
    const firstClusterOption = options.find(
      (opt) => opt.textContent?.trim() === firstClusterName
    );
    expect(firstClusterOption).toBeDefined();

    // click on first cluster option
    await userEvent.click(firstClusterOption!);
    const filteredRows = screen.getAllByRole("row");

    expect(filteredRows).toHaveLength(
      (manager.findCluster(firstClusterName)?.targets.length ?? 0) + 1
    );
  });

  it("should handle multiple selections correctly", async () => {
    render(
      <PrometheusTargetProvider>
        <PopoverSelector {...defaultProps} />
        <TargetList />
      </PrometheusTargetProvider>
    );

    const button = screen.getByRole("combobox");

    await waitFor(() => {
      expect(button).not.toBeDisabled();

      const rows = screen.getAllByRole("row");
      expect(rows).toHaveLength(6);
    });

    await userEvent.click(button);

    const options = await screen.findAllByRole("option");
    expect(options).toHaveLength(2);

    options.forEach(async (option) => await userEvent.click(option));

    const filteredRows = screen.getAllByRole("row");

    // all targets + 2 header rows
    expect(filteredRows).toHaveLength(
      manager.targets.length + manager.clusters.length
    );

    await userEvent.click(button);
    expect(screen.getByText("2 selected")).toBeInTheDocument();
  });
});
