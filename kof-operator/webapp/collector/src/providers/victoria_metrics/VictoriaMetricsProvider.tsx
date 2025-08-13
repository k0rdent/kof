import {
  CollectorMetricsSet,
  PodsMap,
} from "@/components/pages/collectorPage/models";
import { create } from "zustand";
import { CollectorMetricsRecordsManager } from "../collectors_metrics/CollectorsMetricsRecordManager";
import { DefaultProviderState } from "../DefaultProviderState";

export interface Response {
  clusters: Record<string, PodsMap>;
}

export const useVictoriaMetricsState = create<DefaultProviderState>()(
  (set, get) => {
    const metricsHistory = new CollectorMetricsRecordsManager();

    const fetchMetrics = async (): Promise<void> => {
      try {
        set({ isLoading: true, error: undefined });
        const response = await fetch(
          import.meta.env.VITE_VICTORIA_METRICS_URL,
          {
            method: "GET",
          }
        );

        if (!response.ok) {
          throw new Error(`Response status ${response.status}`);
        }

        const json = (await response.json()) as Response;
        const victoriaMetrics = new CollectorMetricsSet(json.clusters);
        metricsHistory.add(victoriaMetrics);
        set({ data: victoriaMetrics, isLoading: false, error: undefined });
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

    const setSelectedPod = (name: string): void => {
      const cluster = get().selectedCluster;
      if (cluster) {
        set({ selectedPod: cluster.getPod(name) });
      }
    };

    fetchMetrics();

    return {
      isLoading: true,
      data: null,
      selectedCluster: null,
      selectedPod: null,
      metricsHistory,
      fetch: fetchMetrics,
      setSelectedCluster,
      setSelectedPod,
    };
  }
);
