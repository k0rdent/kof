import { create } from "zustand";
import {
  ClusterDeployment,
  ClusterDeploymentData,
  ClusterDeploymentSet,
} from "@/components/pages/clusterDeploymentsPage/models";

interface ClusterDeploymentProviderState {
  error?: Error;
  isLoading: boolean;
  data: ClusterDeploymentSet | null;
  selectedCluster: ClusterDeployment | null;
  fetch: () => Promise<void>;
  setSelectedCluster: (name: string) => void;
}

interface ApiResponse {
  cluster_deployments: Record<string, ClusterDeploymentData>;
}

export const useClusterDeploymentsProvider =
  create<ClusterDeploymentProviderState>()((set, get) => {
    const fetchClusters = async (): Promise<void> => {
      try {
        set({ isLoading: true, error: undefined });
        const response = await fetch(
          import.meta.env.VITE_CLUSTER_DEPLOYMENTS_URL,
          {
            method: "GET",
          }
        );

        if (!response.ok) {
          throw new Error(`Response status ${response.status}`);
        }

        const json = (await response.json()) as ApiResponse;
        const map = new ClusterDeploymentSet(json.cluster_deployments);
        set({ data: map, isLoading: false, error: undefined });
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

    fetchClusters();

    return {
      isLoading: true,
      data: null,
      selectedCluster: null,
      fetch: fetchClusters,
      setSelectedCluster,
    };
  });
