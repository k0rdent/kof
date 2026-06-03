import { describe, it, expect } from "vitest";
import { getDuplicatedScrapeUrls } from "../src/utils/target";
import { Target } from "../src/models/PrometheusTarget";

function makeTarget(
  node: string,
  scrapeUrl: string,
  overrides: Partial<Target> = {},
): Target {
  return {
    discoveredLabels: {
      __address__: "127.0.0.1:2379",
      __metrics_path__: "/metrics",
      __scheme__: "https",
      __scrape_interval__: "5s",
      __scrape_timeout__: "4s",
      job: "etcd",
    },
    globalUrl: "https://10.110.3.11:2379/metrics",
    health: "up",
    labels: {
      instance: "10.110.3.11:2379",
      job: "etcd",
    },
    lastError: "",
    lastScrape: new Date("2026-06-03T12:32:02.344194172Z"),
    lastScrapeDuration: 0.044566231,
    scrapeInterval: "5s",
    scrapePool: "etcd",
    scrapeTimeout: "4s",
    scrapeUrl,
    node,
    ...overrides,
  };
}

describe("getDuplicatedScrapeUrls", () => {
  it("excludes the URL when 127.0.0.1 scrapeUrl has targets on different nodes", () => {
    // Provided input: same endpoint scraped from three distinct nodes — not a duplicate
    const targetsMap = {
      "https://127.0.0.1:2379/metrics": [
        makeTarget("mothership-1", "https://127.0.0.1:2379/metrics"),
        makeTarget("mothership-2", "https://127.0.0.1:2379/metrics"),
        makeTarget("mothership-3", "https://127.0.0.1:2379/metrics"),
      ],
    };

    expect(getDuplicatedScrapeUrls(targetsMap)).toEqual([]);
  });

  it("returns the URL when 127.0.0.1 scrapeUrl has multiple targets on the same node", () => {
    const targetsMap = {
      "https://127.0.0.1:2379/metrics": [
        makeTarget("mothership-1", "https://127.0.0.1:2379/metrics"),
        makeTarget("mothership-1", "https://127.0.0.1:2379/metrics"),
        makeTarget("mothership-1", "https://127.0.0.1:2379/metrics"),
      ],
    };

    expect(getDuplicatedScrapeUrls(targetsMap)).toEqual([
      "https://127.0.0.1:2379/metrics",
    ]);
  });

  it("returns the URL when localhost scrapeUrl has multiple targets on the same node", () => {
    const targetsMap = {
      "http://localhost:9090/metrics": [
        makeTarget("node-1", "http://localhost:9090/metrics"),
        makeTarget("node-1", "http://localhost:9090/metrics"),
      ],
    };

    expect(getDuplicatedScrapeUrls(targetsMap)).toEqual([
      "http://localhost:9090/metrics",
    ]);
  });

  it("returns the URL when a non-loopback scrapeUrl has multiple targets on the same node", () => {
    const targetsMap = {
      "https://10.0.0.1:9090/metrics": [
        makeTarget("node-1", "https://10.0.0.1:9090/metrics"),
        makeTarget("node-1", "https://10.0.0.1:9090/metrics"),
      ],
    };

    expect(getDuplicatedScrapeUrls(targetsMap)).toEqual([
      "https://10.0.0.1:9090/metrics",
    ]);
  });

  it("excludes the URL when a non-loopback scrapeUrl has targets on different nodes", () => {
    const targetsMap = {
      "https://10.0.0.1:9090/metrics": [
        makeTarget("node-1", "https://10.0.0.1:9090/metrics"),
        makeTarget("node-2", "https://10.0.0.1:9090/metrics"),
      ],
    };

    expect(getDuplicatedScrapeUrls(targetsMap)).toEqual([]);
  });

  it("returns the URL when at least two targets share the same node even if one target has a different node", () => {
    const targetsMap = {
      "https://127.0.0.1:2379/metrics": [
        makeTarget("mothership-1", "https://127.0.0.1:2379/metrics"),
        makeTarget("mothership-1", "https://127.0.0.1:2379/metrics"),
        makeTarget("mothership-2", "https://127.0.0.1:2379/metrics"),
      ],
    };

    expect(getDuplicatedScrapeUrls(targetsMap)).toEqual([
      "https://127.0.0.1:2379/metrics",
    ]);
  });

  it("excludes URLs with only a single target", () => {
    const targetsMap = {
      "https://127.0.0.1:2379/metrics": [
        makeTarget("mothership-1", "https://127.0.0.1:2379/metrics"),
      ],
      "https://10.0.0.1:9090/metrics": [
        makeTarget("node-1", "https://10.0.0.1:9090/metrics"),
      ],
    };

    expect(getDuplicatedScrapeUrls(targetsMap)).toEqual([]);
  });

  it("returns an empty array for an empty targetsMap", () => {
    expect(getDuplicatedScrapeUrls({})).toEqual([]);
  });
});
