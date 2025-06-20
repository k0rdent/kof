import { PrometheusTargetsManager } from "../src/models/PrometheusTargetsManager";
import { Cluster } from "../src/models/Cluster";
import { HealthFilter } from "../src/components/features/HealthSelector";
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
    const filter = HealthFilter(["up"]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(2);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);

    const filteredTargets = filteredClusters.flatMap((c) => c.targets);
    expect(filteredTargets.every((target) => target.health === "up")).toBe(
      true
    );
  });

  it("should return clusters with only 'down' targets", () => {
    const filter = HealthFilter(["down"]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(1);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);

    const filteredTargets = filteredClusters.flatMap((c) => c.targets);
    expect(filteredTargets.every((target) => target.health === "down")).toBe(
      true
    );
  });

  it("should return clusters with only 'unknown' targets", () => {
    const filter = HealthFilter(["unknown"]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(1);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);

    const filteredTargets = filteredClusters.flatMap((c) => c.targets);
    expect(filteredTargets.every((target) => target.health === "unknown")).toBe(
      true
    );
  });

  it("should return clusters with targets matching any of the specified states", () => {
    const filter = HealthFilter(["up", "down"]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(1);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(3);
    expect(filteredClusters.every((cluster) => cluster.hasNodes)).toBe(true);

    const filteredTargets = filteredClusters.flatMap((c) => c.targets);
    expect(
      filteredTargets.every(
        (target) => target.health === "down" || target.health === "up"
      )
    ).toBe(true);
  });

  it("should return clusters with all health states when all states are specified", () => {
    const filter = HealthFilter(["up", "down", "unknown"]);
    const filteredClusters: Cluster[] = filter(clusters);

    expect(filteredClusters.length).toBe(2);
    expect(filteredClusters.flatMap((c) => c.targets).length).toBe(4);
    expect(filteredClusters.every((cluster: Cluster) => cluster.hasNodes)).toBe(
      true
    );
  });

  it("should handle empty clusters array", () => {
    const filter = HealthFilter(["up"]);
    const result = filter([]);

    expect(result).toHaveLength(0);
  });
});
