import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
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
    render(
      <PrometheusTargetProvider>
        <DuplicateTargetsAlert clusterName="aws-ue2-test-1" />
      </PrometheusTargetProvider>
    );

    // The fake data has the same scrape URL appearing on different nodes, which is
    // not considered a duplication (e.g. loopback addresses scraped per-node).
    // The component should render without showing the duplicate alert.
    expect(screen.queryByText("Some targets are duplicated and scraping the same URL")).not.toBeInTheDocument();
  });
});
