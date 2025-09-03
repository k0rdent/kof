import { create } from "zustand";
import {
  ClusterSummariesSet,
  ClusterSummary,
  ClusterSummaryData,
} from "@/components/pages/cluster_summaries_page/models";

interface ClusterSummariesProviderState {
  error?: Error;
  isLoading: boolean;
  data: ClusterSummariesSet | null;
  selectedSummary: ClusterSummary | null;
  fetch: () => Promise<void>;
  setSelectedSummary: (name: string) => void;
}

interface ApiResponse {
  cluster_summaries: Record<string, ClusterSummaryData>;
}

export const useClusterSummariesProvider =
  create<ClusterSummariesProviderState>()((set, get) => {
    const fetchClusters = async (): Promise<void> => {
      try {
        set({ isLoading: true, error: undefined });
        const response = await fetch(
          import.meta.env.VITE_CLUSTER_SUMMARIES_URL,
          {
            method: "GET",
          }
        );

        if (!response.ok) {
          throw new Error(`Response status ${response.status}`);
        }

        const json = (await response.json()) as ApiResponse;
        const map = new ClusterSummariesSet({
          clusterSummaries: json.cluster_summaries,
        });
        set({ data: map, isLoading: false, error: undefined });
      } catch (e) {
        set({ data: null, error: e as Error, isLoading: false });
      }
    };

    const setSelectedSummary = (name: string): void => {
      const data = get().data;
      if (data) {
        set({ selectedSummary: data.getSummary(name) });
      }
    };

    fetchClusters();

    return {
      isLoading: true,
      data: null,
      selectedSummary: null,
      fetch: fetchClusters,
      setSelectedSummary,
    };
  });
