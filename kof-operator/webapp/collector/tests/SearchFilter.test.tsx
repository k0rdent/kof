import { describe, it, expect, beforeEach } from "vitest";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import { SearchFilter } from "../src/components/features/SearchBar";
import { fakeData } from "./fake_data/fake_response";


describe("Search filter", () => {
  let prometheusTargetsManager: PrometheusTargetsManager;

  beforeEach(() => {
    prometheusTargetsManager = new PrometheusTargetsManager({
      clusters: fakeData.clusters,
    });
  });

  it("should filter clusters using scrapeUrl match", () => {
    const searchFilter = SearchFilter("http://10.244.67.136:15090/metrics");
    const filteredClusters = searchFilter(prometheusTargetsManager.clusters);

    expect(filteredClusters).toHaveLength(1);
    expect(filteredClusters[0].name).toBe("aws-ue2-test-1");
  });

  it("should filter clusters using label match", () => {
    const searchFilter = SearchFilter("node-exporter");
    const filteredClusters = searchFilter(prometheusTargetsManager.clusters);

    //Both has the same label
    expect(filteredClusters).toHaveLength(2);
  });

  it("should filter clusters using discoveredLabels value match", () => {
    const searchFilter = SearchFilter("kcm");
    const filteredClusters = searchFilter(prometheusTargetsManager.clusters);

    // Should find the cluster where discoveredLabels contains 'kof'
    expect(filteredClusters).toHaveLength(1);
    expect(filteredClusters[0].name).toBe("aws-ue2-test-2");
  });

  it("should return empty array when no matches found", () => {
    const searchFilter = SearchFilter("empty_value");
    const filteredClusters = searchFilter(prometheusTargetsManager.clusters);

    expect(filteredClusters).toHaveLength(0);
  });

  it("should return all clusters when search value is empty", () => {
    const searchFilter = SearchFilter("");
    const filteredClusters = searchFilter(prometheusTargetsManager.clusters);

    expect(filteredClusters).toHaveLength(2);
    expect(filteredClusters[0].name).toBe("aws-ue2-test-1");
    expect(filteredClusters[1].name).toBe("aws-ue2-test-2");
  });

  it("should not be case sensitive", () => {
    const searchFilter = SearchFilter("KOF");
    const filteredClusters = searchFilter(prometheusTargetsManager.clusters);

    expect(filteredClusters).toHaveLength(1);
  });

  it("should filter by partial matches in labels", () => {
    const searchFilter = SearchFilter("node");
    const filteredClusters = searchFilter(prometheusTargetsManager.clusters);

    expect(filteredClusters).toHaveLength(2);
  });
});
