import {
  Cluster,
  CollectorMetricsSet,
  Pod,
} from "@/components/pages/collectorPage/models";
import { create } from "zustand";
import { CollectorMetricsRecordsManager } from "../collectors_metrics/CollectorsMetricsRecordManager"

interface VictoriaMetricsState{
  error?: Error;
  isLoading: boolean;
  data: CollectorMetricsSet | null;
  selectedCluster: Cluster | null;
  selectedPod: Pod | null;
  metricsHistory: CollectorMetricsRecordsManager;
  fetch: () => void;
  setSelectedCluster: (name: string) => void;
  setSelectedPod: (name: string) => void;
};

export const useVictoriaMetricsState = create<VictoriaMetricsState>()(
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

        const json = await response.json();
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
