import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import HealthSelector from "../src/components/features/HealthSelector";
import { ClustersData } from "../src/models/Cluster";
import { fakeData } from "./fake_data/fake_response";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";
import TargetList from "../src/components/features/TargetsList";
import { userEvent } from "@testing-library/user-event";

describe("Health selector", () => {
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

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should render health selector elements", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
      </PrometheusTargetProvider>
    );

    expect(screen.getByRole("up-checkbox")).toBeInTheDocument();
    expect(screen.getByRole("unknown-checkbox")).toBeInTheDocument();
    expect(screen.getByRole("down-checkbox")).toBeInTheDocument();

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

    const upCheckbox = screen.getByRole("up-checkbox") as HTMLInputElement;
    expect(upCheckbox).toBeInTheDocument();
    expect(upCheckbox.disabled).toBe(true);

    const unknownCheckbox = screen.getByRole(
      "unknown-checkbox"
    ) as HTMLInputElement;
    expect(unknownCheckbox).toBeInTheDocument();
    expect(unknownCheckbox.disabled).toBe(true);

    const downCheckbox = screen.getByRole("down-checkbox") as HTMLInputElement;
    expect(downCheckbox).toBeInTheDocument();
    expect(downCheckbox.disabled).toBe(true);

    await waitFor(() => {
      expect(upCheckbox.disabled).toBe(false);
      expect(unknownCheckbox.disabled).toBe(false);
      expect(downCheckbox.disabled).toBe(false);
    });
  });

  it("should display correct target counts for each health state", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
      </PrometheusTargetProvider>
    );

    await waitFor(() => {
      const upCheckbox = screen.getByRole("up-checkbox") as HTMLInputElement;
      expect(upCheckbox).toBeInTheDocument();
      expect(upCheckbox.disabled).toBe(false);

      const unknownCheckbox = screen.getByRole(
        "unknown-checkbox"
      ) as HTMLInputElement;
      expect(unknownCheckbox).toBeInTheDocument();
      expect(unknownCheckbox.disabled).toBe(false);

      const downCheckbox = screen.getByRole(
        "down-checkbox"
      ) as HTMLInputElement;
      expect(downCheckbox).toBeInTheDocument();
      expect(downCheckbox.disabled).toBe(false);

      expect(screen.getByText(`1 Unknown`)).toBeInTheDocument();
      expect(screen.getByText(`1 Down`)).toBeInTheDocument();
      expect(screen.getByText(`2 Up`)).toBeInTheDocument();
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
      const upCheckbox = screen.getByRole("up-checkbox") as HTMLInputElement;
      expect(upCheckbox).toBeInTheDocument();
      expect(upCheckbox.disabled).toBe(false);

      const unknownCheckbox = screen.getByRole(
        "unknown-checkbox"
      ) as HTMLInputElement;
      expect(unknownCheckbox).toBeInTheDocument();
      expect(unknownCheckbox.disabled).toBe(false);

      const downCheckbox = screen.getByRole(
        "down-checkbox"
      ) as HTMLInputElement;
      expect(downCheckbox).toBeInTheDocument();
      expect(downCheckbox.disabled).toBe(false);
    });

    const upCheckbox = screen.getByRole("up-checkbox");

    await userEvent.click(upCheckbox);
    const filteredRows = screen.getAllByRole("row");

    // 1 header + 2 target
    expect(filteredRows).toHaveLength(3);
  });

  it("should handle multiple checkbox selections", async () => {
    render(
      <PrometheusTargetProvider>
        <HealthSelector />
        <TargetList />
      </PrometheusTargetProvider>
    );

    await waitFor(() => {
      const upCheckbox = screen.getByRole("up-checkbox") as HTMLInputElement;
      expect(upCheckbox).toBeInTheDocument();
      expect(upCheckbox.disabled).toBe(false);

      const unknownCheckbox = screen.getByRole(
        "unknown-checkbox"
      ) as HTMLInputElement;
      expect(unknownCheckbox).toBeInTheDocument();
      expect(unknownCheckbox.disabled).toBe(false);

      const downCheckbox = screen.getByRole(
        "down-checkbox"
      ) as HTMLInputElement;
      expect(downCheckbox).toBeInTheDocument();
      expect(downCheckbox.disabled).toBe(false);
    });

    const upCheckbox = screen.getByRole("up-checkbox");
    const downCheckbox = screen.getByRole("down-checkbox");

    await userEvent.click(upCheckbox);
    await userEvent.click(downCheckbox);

    const filteredRows = screen.getAllByRole("row");

    // 1 header + 3 target
    expect(filteredRows).toHaveLength(4);
  });
});
