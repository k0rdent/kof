import { FilterFunction } from "@/providers/prometheus/PrometheusTargetsProvider";
import { PrometheusTargetsManager } from "./PrometheusTargetsManager";
import { Cluster } from "./Cluster";

export interface PrometheusContext {
    data: PrometheusTargetsManager | null
    filteredData: Cluster[] | null
    addFilter: (name: string, filterFn: FilterFunction) => string;
    removeFilter: (id: string) => void;
    clearFilters: () => void;
    loading: boolean;
    error: Error | null;
    fetchPrometheusTargets: () => Promise<void>;
}

export interface PodResponse {
    data: PrometheusTargetData
    success: boolean
}

export interface PrometheusTargetData {
    activeTargets: Target[]
    droppedTargetCounts: Map<string, string>
    droppedTargets: Target[]
}

export interface Target {
    discoveredLabels: Record<string, string>
    globalUrl: string
    health: string
    labels: Record<string, string>
    lastError?: string
    lastScrape: Date
    lastScrapeDuration: number
    scrapeInterval: string
    scrapePool: string
    scrapeTimeout: string
    scrapeUrl: string
}