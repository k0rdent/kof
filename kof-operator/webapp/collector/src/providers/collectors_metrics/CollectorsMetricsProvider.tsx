import { ClustersSet } from "@/components/pages/collectorPage/models";
import { create } from "zustand";
import { DefaultProviderState } from "../DefaultProviderState";
import { MetricsRecordsService } from "../collectors_metrics/CollectorsMetricsRecordManager";

export const useCollectorMetricsState = create<DefaultProviderState>()(
  (set, get) => {
    const metricsHistory = MetricsRecordsService;

    const fetchMetrics = async (): Promise<void> => {
      try {
        set({ isLoading: true, error: undefined });
        const response = await fetch(
          import.meta.env.VITE_COLLECTOR_METRICS_URL,
          {
            method: "GET",
          }
        );

        if (!response.ok) {
          throw new Error(`Response status ${response.status}`);
        }

        const json = await response.json();
        const collectorsMetrics = new ClustersSet(json.clusters);
        MetricsRecordsService.add(collectorsMetrics);
        set({ data: collectorsMetrics, isLoading: false, error: undefined });
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
      setSelectedPod: setSelectedCollector,
    };
  }
);
