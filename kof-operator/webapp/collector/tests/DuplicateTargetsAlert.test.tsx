import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import DuplicateTargetsAlert from "../src/components/features/DuplicateTargetsAlert";
import { ClustersData } from "../src/models/Cluster";
import { fakeDuplicatedTargetsData } from "./fake_data/fake_response";
import PrometheusTargetProvider from "../src/providers/prometheus/PrometheusTargetsProvider";

window.HTMLElement.prototype.scrollIntoView = function () { };

describe("Duplicate targets alert", () => {
  const fakeClusters: ClustersData = fakeDuplicatedTargetsData;
  const mockOnSelectionChange = vi.fn();

  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(fakeClusters),
      })
    );
    mockOnSelectionChange.mockClear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should render successfully", async () => {
    await act(() => render(
      <PrometheusTargetProvider>
        <DuplicateTargetsAlert clusterName="aws-ue2-test-1" />
      </PrometheusTargetProvider>
    ));

    expect(screen.getByText("Some targets are duplicated and scraping the same URL")).toBeInTheDocument();
    expect(screen.getByText("Scrape URL: http://10.244.67.136:15090/metrics")).toBeInTheDocument();
  });
});
