import { ClustersData } from "../../src/models/Cluster";


export const fakeData: ClustersData = {
    clusters: {
        "aws-ue2-test-1": {
            nodes: {
                "aws-ue2-test-1-cp-0": {
                    pods: {
                        "kof-collectors-node-exporter-collector-g8646": {
                            success: true,
                            data: {
                                activeTargets: [
                                    {
                                        discoveredLabels: {
                                            __address__: "10.244.67.136:15090",
                                            __meta_kubernetes_namespace: "kof",
                                        },
                                        globalUrl: "http://10.244.67.136:15090",
                                        scrapePool: "kof-pool",
                                        labels: { job: "node-exporter", app: "kof" },
                                        scrapeUrl: "http://10.244.67.136:15090/metrics",
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapeTimeout: "10s",
                                        lastError: undefined,
                                        health: "up",
                                    },
                                ],
                                droppedTargetCounts: new Map<string, string>(),
                                droppedTargets: [],
                            },
                        },
                    },
                },
                "aws-ue2-test-1-worker-1": {
                    pods: {
                        "prometheus-collector-worker": {
                            data: {
                                activeTargets: [
                                    {
                                        discoveredLabels: {
                                            __meta_kubernetes_namespace: "monitoring",
                                        },
                                        globalUrl: "http://10.244.67.140:9090",
                                        labels: { job: "prometheus", app: "monitoring" },
                                        scrapeUrl: "http://10.244.67.140:9090/metrics",
                                        scrapePool: "prometheus-pool",
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapeTimeout: "10s",
                                        lastError: undefined,
                                        health: "down",
                                    },
                                ],
                            },
                        },
                    },
                },
                "aws-ue2-test-1-worker-2": {
                    pods: {
                        "grafana-collector": {
                            data: {
                                activeTargets: [
                                    {
                                        discoveredLabels: {
                                            __meta_kubernetes_namespace: "grafana",
                                        },
                                        globalUrl: "http://10.244.67.141:3000",
                                        labels: { job: "grafana", app: "grafana" },
                                        scrapeUrl: "http://10.244.67.141:3000/metrics",
                                        scrapePool: "grafana-pool",
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapeTimeout: "10s",
                                        lastError: undefined,
                                        health: "up",
                                    },
                                ],
                            },
                        },
                    },
                },
            },
        },
        "aws-ue2-test-2": {
            nodes: {
                "aws-ue2-test-2-cp-0": {
                    pods: {
                        "kof-collectors-node-exporter-collector-g8643126": {
                            success: true,
                            data: {
                                activeTargets: [
                                    {
                                        discoveredLabels: {
                                            __meta_kubernetes_namespace: "kcm",
                                        },
                                        globalUrl: "http://10.123.69.163:15090",
                                        health: "unknown",
                                        labels: { job: "node-exporter", app: "kcm" },
                                        lastError: undefined,
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapePool: "kcm-pool",
                                        scrapeTimeout: "10s",
                                        scrapeUrl: "http://10.123.69.163/metrics",
                                    },
                                ],
                                droppedTargetCounts: new Map<string, string>(),
                                droppedTargets: [],
                            },
                        },
                    },
                },
            },
        },
    },
};

export const fakeDuplicatedTargetsData: ClustersData = {
    clusters: {
        "aws-ue2-test-1": {
            nodes: {
                "aws-ue2-test-1-cp-0": {
                    pods: {
                        "kof-collectors-node-exporter-collector-g8646": {
                            success: true,
                            data: {
                                activeTargets: [
                                    {
                                        discoveredLabels: {
                                            __address__: "10.244.67.136:15090",
                                            __meta_kubernetes_namespace: "kof",
                                        },
                                        globalUrl: "http://10.244.67.136:15090",
                                        scrapePool: "kof-pool",
                                        labels: { job: "node-exporter", app: "kof" },
                                        scrapeUrl: "http://10.244.67.136:15090/metrics",
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapeTimeout: "10s",
                                        lastError: undefined,
                                        health: "up",
                                    },
                                    {
                                        discoveredLabels: {
                                            __address__: "127.0.0.1:15091",
                                            __meta_kubernetes_namespace: "kof",
                                        },
                                        globalUrl: "http://127.0.0.1:15091",
                                        scrapePool: "kof-pool",
                                        labels: { job: "node-exporter", app: "kof" },
                                        scrapeUrl: "http://127.0.0.1:15091/metrics",
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapeTimeout: "10s",
                                        lastError: undefined,
                                        health: "up",
                                    },
                                ],
                                droppedTargetCounts: new Map<string, string>(),
                                droppedTargets: [],
                            },
                        },
                    },
                },
                "aws-ue2-test-1-worker-1": {
                    pods: {
                        "prometheus-collector-worker": {
                            success: true,
                            data: {
                                activeTargets: [
                                    {
                                        discoveredLabels: {
                                            __address__: "10.244.67.136:15090",
                                            __meta_kubernetes_namespace: "kof",
                                        },
                                        globalUrl: "http://10.244.67.136:15090",
                                        scrapePool: "kof-pool",
                                        labels: { job: "node-exporter", app: "kof" },
                                        scrapeUrl: "http://10.244.67.136:15090/metrics",
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapeTimeout: "10s",
                                        lastError: undefined,
                                        health: "up",
                                    },
                                    {
                                        discoveredLabels: {
                                            __address__: "127.0.0.1:15091",
                                            __meta_kubernetes_namespace: "kof",
                                        },
                                        globalUrl: "http://127.0.0.1:15091",
                                        scrapePool: "kof-pool",
                                        labels: { job: "node-exporter", app: "kof" },
                                        scrapeUrl: "http://127.0.0.1:15091/metrics",
                                        lastScrape: new Date("2025-05-28T10:00:00Z"),
                                        lastScrapeDuration: 0.123,
                                        scrapeInterval: "15s",
                                        scrapeTimeout: "10s",
                                        lastError: undefined,
                                        health: "up",
                                    },
                                ],
                            },
                        },
                    },
                },
            },
        },
    },
};