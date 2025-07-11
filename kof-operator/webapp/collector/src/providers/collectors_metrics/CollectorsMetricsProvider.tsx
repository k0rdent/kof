import {
  Cluster,
  CollectorMetrics,
  Pod,
} from "@/components/pages/collectorPage/models";
import { create } from "zustand";

type CollectorMetricsState = {
  error?: Error;
  isLoading: boolean;
  data: CollectorMetrics | null;
  selectedCluster: Cluster | null;
  selectedCollector: Pod | null;
  fetch: () => void;
  setSelectedCluster: (name: string) => void;
  setSelectedCollector: (name: string) => void;
};

export const useCollectorMetricsState = create<CollectorMetricsState>()(
  (set, get) => {
    const fetchData = async () => {
      set({ isLoading: true, error: undefined });
      try {
        const response = await fetch(
          import.meta.env.VITE_COLLECTOR_METRICS_URL,
          {
            method: "GET",
          }
        );
        if (!response.ok) {
          set({
            isLoading: false,
            error: new Error(`Response status ${response.status}`),
          });
          return;
        }
        const json = await response.json();
        const data: CollectorMetrics = new CollectorMetrics(json.clusters);
        set({ isLoading: false, error: undefined, data });
      } catch (e) {
        set({ isLoading: false, error: e as Error, data: null });
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

    fetchData();

    return {
      isLoading: true,
      data: null,
      selectedCluster: null,
      selectedCollector: null,
      fetch: fetchData,
      setSelectedCluster,
      setSelectedCollector,
    };
  }
);
