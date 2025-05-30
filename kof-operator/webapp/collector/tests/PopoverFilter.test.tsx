import { describe, it, expect, beforeEach } from "vitest";
import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import {
  PopoverClusterFilter,
  PopoverNodeFilter,
} from "../src/components/features/PopoverSelector";
import { fakeData } from "./fake_data/fake_response";
import { Cluster } from "../src/models/Cluster";

describe("Popover cluster filter", () => {
  let prometheusTargetsManager: PrometheusTargetsManager;
  let clusters: any[];

  beforeEach(() => {
    prometheusTargetsManager = new PrometheusTargetsManager({
      clusters: fakeData.clusters,
    });
    clusters = prometheusTargetsManager.clusters;
  });

  it("should return only the matching cluster", () => {
    const filter = PopoverClusterFilter(["aws-ue2-test-1"]);
    const result = filter(clusters);

    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("aws-ue2-test-1");
  });

  it("should return all matching clusters", () => {
    const filter = PopoverClusterFilter(["aws-ue2-test-1", "aws-ue2-test-2"]);
    const result = filter(clusters);

    expect(result).toHaveLength(2);
    expect(result.map((cluster: Cluster) => cluster.name)).toContain(
      "aws-ue2-test-1"
    );
    expect(result.map((cluster: Cluster) => cluster.name)).toContain(
      "aws-ue2-test-2"
    );
  });

  it("should return empty array for completely non existent names", () => {
    const filter = PopoverClusterFilter(["non-existent-cluster"]);
    const result = filter(clusters);

    expect(result).toHaveLength(0);
  });
});

describe("Popover node filter", () => {
  let prometheusTargetsManager: PrometheusTargetsManager;
  let clusters: any[];

  beforeEach(() => {
    prometheusTargetsManager = new PrometheusTargetsManager({
      clusters: fakeData.clusters,
    });
    clusters = prometheusTargetsManager.clusters;
  });

  it("should return clusters containing only the matching node", () => {
    const filter = PopoverNodeFilter(["aws-ue2-test-1-worker-1"]);
    const result = filter(clusters);

    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("aws-ue2-test-1");
    expect(result[0].hasNodes).toBe(true);
  });

  it("should return empty array when node doesn't exist in any cluster", () => {
    const filter = PopoverNodeFilter(["unknown-node"]);
    const result = filter(clusters);

    expect(result).toHaveLength(0);
  });

  it("should return clusters with all matching nodes from same cluster", () => {
    const filter = PopoverNodeFilter([
      "aws-ue2-test-1-cp-0",
      "aws-ue2-test-1-worker-1",
    ]);
    const result = filter(clusters);

    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("aws-ue2-test-1");
    expect(result[0].hasNodes).toBe(true);
  });

  it("should return multiple clusters when nodes are from different clusters", () => {
    const filter = PopoverNodeFilter([
      "aws-ue2-test-1-worker-1",
      "aws-ue2-test-2-cp-0",
    ]);
    const result = filter(clusters);

    expect(result).toHaveLength(2);
    expect(result.map((cluster: Cluster) => cluster.name)).toContain(
      "aws-ue2-test-1"
    );
    expect(result.map((cluster: Cluster) => cluster.name)).toContain(
      "aws-ue2-test-2"
    );
    expect(result.every((cluster: Cluster) => cluster.hasNodes)).toBe(true);
  });
});
