import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import HealthSelector from "../src/components/features/HealthSelector";
import { ClustersData } from "../src/models/Cluster";
import { fakeData } from "./fake_data/fake_response";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";
import TargetList from "../src/components/features/TargetsList";
import { userEvent } from "@testing-library/user-event";

describe("Health selector", () => {
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

  it("should render health selector elements", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
      </PrometheusTargetProvider>
    );

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);

    expect(screen.getByText("0 Down")).toBeInTheDocument();
    expect(screen.getByText("0 Up")).toBeInTheDocument();
    expect(screen.getByText("0 Unknown")).toBeInTheDocument();
  });

  it("should disable checkboxes when loading", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
      </PrometheusTargetProvider>
    );

    const checkboxes = screen.getAllByRole("checkbox") as HTMLInputElement[];

    checkboxes.forEach((checkbox) => {
      expect(checkbox.disabled).toBe(true);
    });

    await waitFor(() => {
      checkboxes.forEach((checkbox) => {
        expect(checkbox.disabled).toBe(false);
      });
    });
  });

  it("should display correct target counts for each health state", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
      </PrometheusTargetProvider>
    );

    const upCount = manager.targets.filter(
      (target) => target.health === "up"
    ).length;
    const downCount = manager.targets.filter(
      (target) => target.health === "down"
    ).length;
    const unknownCount = manager.targets.filter(
      (target) => target.health === "unknown"
    ).length;

    await waitFor(() => {
      const checkboxes = screen.getAllByRole("checkbox") as HTMLInputElement[];
      checkboxes.forEach((checkbox) => {
        expect(checkbox.disabled).toBe(false);
      });

      expect(screen.getByText(`${unknownCount} Unknown`)).toBeInTheDocument();
      expect(screen.getByText(`${downCount} Down`)).toBeInTheDocument();
      expect(screen.getByText(`${upCount} Up`)).toBeInTheDocument();
    });
  });

  it("should handle single checkbox selection", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
        <TargetList />
      </PrometheusTargetProvider>
    );

    await waitFor(() => {
      const checkboxes = screen.getAllByRole("checkbox") as HTMLInputElement[];
      checkboxes.forEach((checkbox) => {
        expect(checkbox.disabled).toBe(false);
      });
    });

    const upCount = manager.targets.filter(
      (target) => target.health === "up"
    ).length;

    const checkboxes = screen.getAllByRole("checkbox");
    const upCheckbox = checkboxes[2];

    await userEvent.click(upCheckbox);
    const filteredRows = screen.getAllByRole("row");

    // 1 header + 2 target
    expect(filteredRows).toHaveLength(upCount + 1);
  });

  it("should handle multiple checkbox selections", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
        <TargetList />
      </PrometheusTargetProvider>
    );

    await waitFor(() => {
      const checkboxes = screen.getAllByRole("checkbox") as HTMLInputElement[];
      checkboxes.forEach((checkbox) => {
        expect(checkbox.disabled).toBe(false);
      });
    });

    const checkboxes = screen.getAllByRole("checkbox");
    const upCheckbox = checkboxes[2];
    const downCheckbox = checkboxes[1];

    const upCount = manager.targets.filter(
      (target) => target.health === "up"
    ).length;

    const downCount = manager.targets.filter(
      (target) => target.health === "down"
    ).length;

    await userEvent.click(upCheckbox);
    await userEvent.click(downCheckbox);

    const filteredRows = screen.getAllByRole("row");

    // 1 header + 3 target
    expect(filteredRows).toHaveLength(upCount + downCount + 1);
  });
});
