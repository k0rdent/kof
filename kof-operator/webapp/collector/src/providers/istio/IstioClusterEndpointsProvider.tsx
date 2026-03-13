import { create } from "zustand";
import { ClusterConnectivity, ClusterEndpoints } from "./IstioMeshProvider";

const storeKey = (clusterName: string, namespace: string): string =>
  `${clusterName}:${namespace}`;

type ClusterEndpointsState = {
  data: Map<string, ClusterConnectivity>;
  loading: Set<string>;
  errors: Map<string, Error>;
  fetchForCluster: (clusterName: string, namespace: string) => Promise<void>;
};

export const useClusterEndpoints = create<ClusterEndpointsState>()((set, get) => ({
  data: new Map(),
  loading: new Set(),
  errors: new Map(),

  fetchForCluster: async (clusterName: string, namespace: string) => {
    const key = storeKey(clusterName, namespace);
    if (get().loading.has(key)) return;

    set((s) => {
      const loading = new Set(s.loading);
      loading.add(key);
      const errors = new Map(s.errors);
      errors.delete(key);
      return { ...s, loading, errors };
    });

    try {
      const url = new URL(import.meta.env.VITE_ISTIO_MESH_ENDPOINTS_URL, window.location.href);
      url.searchParams.set("cluster", clusterName);
      if (namespace) url.searchParams.set("namespace", namespace);

      const response = await fetch(url.toString(), { method: "GET" });
      if (!response.ok) throw new Error(`Response status ${response.status}`);

      const json: { endpoints: ClusterEndpoints[] } = await response.json();
      const connectivity: ClusterConnectivity = json.endpoints?.[0]?.endpoints ?? {
        sourceCluster: clusterName,
        sourceClusterNamespace: namespace,
        remoteClusters: [],
      };

      set((s) => {
        const data = new Map(s.data);
        data.set(key, connectivity);
        const loading = new Set(s.loading);
        loading.delete(key);
        return { ...s, data, loading };
      });
    } catch (e) {
      set((s) => {
        const errors = new Map(s.errors);
        errors.set(key, e as Error);
        const loading = new Set(s.loading);
        loading.delete(key);
        return { ...s, errors, loading };
      });
    }
  },
}));
