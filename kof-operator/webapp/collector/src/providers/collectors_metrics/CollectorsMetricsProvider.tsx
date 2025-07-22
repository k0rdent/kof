import {
  Cluster,
  CollectorMetricsSet,
  Pod,
} from "@/components/pages/collectorPage/models";
import { create } from "zustand";
import { CollectorMetricsRecordsManager } from "./CollectorsMetricsRecordManager";

export interface LocalStorageMetric {
  timestamp: number;
  data: unknown;
}

type CollectorMetricsState = {
  error?: Error;
  isLoading: boolean;
  data: CollectorMetricsSet | null;
  selectedCluster: Cluster | null;
  selectedCollector: Pod | null;
  metricsHistory: CollectorMetricsRecordsManager;
  fetch: (quiet: boolean) => void;
  setSelectedCluster: (name: string) => void;
  setSelectedCollector: (name: string) => void;
};

export const useCollectorMetricsState = create<CollectorMetricsState>()(
  (set, get) => {
    const metricsHistory = new CollectorMetricsRecordsManager();

    const fetchData = async (): Promise<CollectorMetricsSet> => {
      const response = await fetch(import.meta.env.VITE_COLLECTOR_METRICS_URL, {
        method: "GET",
      });

      if (!response.ok) {
        throw new Error(`Response status ${response.status}`);
      }

      const json = await response.json();
      const collectorsMetrics = new CollectorMetricsSet(json.clusters);
      metricsHistory.add(collectorsMetrics);

      return collectorsMetrics;
    };

    const fetchMetrics = async (quiet = false) => {
      if (!quiet) set({ isLoading: true, error: undefined });
      try {
        const data = await fetchData();
        set({ data, isLoading: false, error: undefined });
      } catch (e) {
        set({ data: null, error: e as Error, isLoading: false });
      }
    };

    const setSelectedCluster = (name: string): void => {
      const data = get().data;
      if (data) {
        set({ selectedCluster: data.getCluster(name) });
      }
    };

    const setSelectedCollector = (name: string): void => {
      const cluster = get().selectedCluster;
      if (cluster) {
        set({ selectedCollector: cluster.getPod(name) });
      }
    };

    fetchMetrics();

    return {
      isLoading: true,
      data: null,
      selectedCluster: null,
      selectedCollector: null,
      metricsHistory,
      fetch: fetchMetrics,
      setSelectedCluster,
      setSelectedCollector,
    };
  }
);
