import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import { Cluster } from "../src/models/Cluster";
import { State, HealthFilter } from "../src/components/features/HealthSelector";
import { fakeData } from "./fake_data/fake_response";
import { describe, it, expect, beforeEach } from "vitest";

describe("Health filter", () => {
  let prometheusTargetsManager: PrometheusTargetsManager;
  let clusters: Cluster[];

  beforeEach(() => {
    prometheusTargetsManager = new PrometheusTargetsManager({
      clusters: fakeData.clusters,
    });
    clusters = prometheusTargetsManager.clusters;
  });

  it("should return all data unchanged", () => {
    const filter = HealthFilter([]);
    const result = filter(clusters);

    expect(result).toEqual(clusters);
    expect(result).toHaveLength(clusters.length);
  });

  it("should return clusters with only 'up' targets", () => {
    const filter = HealthFilter(["up" as State]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(2);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);
  });

  it("should return clusters with only 'down' targets", () => {
    const filter = HealthFilter(["down" as State]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(1);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);
  });

  it("should return clusters with only 'unknown' targets", () => {
    const filter = HealthFilter(["unknown" as State]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(1);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);
  });

  it("should return clusters with targets matching any of the specified states", () => {
    const filter = HealthFilter(["up", "down"] as State[]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(3);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);
  });

  it("should return clusters with all health states when all states are specified", () => {
    const filter = HealthFilter(["up", "down", "unknown"] as State[]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(2);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(4);
    expect(filteredClusters.every((cluster: Cluster) => cluster.hasNodes)).toBe(
      true
    );
  });

  it("should handle empty clusters array", () => {
    const filter = HealthFilter(["up"] as State[]);
    const result = filter([]);

    expect(result).toHaveLength(0);
  });
});
