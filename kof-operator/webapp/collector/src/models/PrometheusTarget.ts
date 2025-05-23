import { FilterFunction } from "@/providers/prometheus/PrometheusTargetsProvider";

export interface PrometheusContext {
    data: PrometheusTargets | null
    filteredData: PrometheusTargets | null
    addFilter: (name: string, filterFn: FilterFunction) => string;
    removeFilter: (id: string) => void;
    clearFilters: () => void;
    loading: boolean;
    error: string | null;
    fetchPrometheusTargets: () => Promise<void>;
}

export interface PrometheusTargets {
   clusters: Cluster[]
}

export interface Cluster {
    name: string
    nodes: Node[]
}

export interface Node {
    name: string
    pods: Pod[]
}

export interface Pod {
    name: string
    response: PodResponse
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