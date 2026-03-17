import { create } from "zustand";

export interface MeshNode {
  id: string;
  name: string;
  namespace: string;
  role: string;
}

export interface MeshLink {
  secretName: string;
  source: string;
  target: string;
}

export interface ServiceEndpoint {
  serviceFqdn: string;
  namespace: string;
  workloadName: string;
  addresses: string[];
  port: number;
  serviceAccount: string;
  tlsMode: string;
  healthy: boolean;
}

export interface ConnectedCluster {
  clusterId: string;
  services: ServiceEndpoint[];
}

export interface ClusterConnectivity {
  sourceCluster: string;
  sourceClusterNamespace: string;
  connectedClusters: ConnectedCluster[];
}

export interface ClusterEndpoints {
  cluster: string;
  endpoints: ClusterConnectivity;
}

export interface MeshGraph {
  nodes: MeshNode[];
  links: MeshLink[];
}

type IstioMeshState = {
  isLoading: boolean;
  data: MeshGraph | null;
  error: Error | null;
  fetch: () => Promise<void>;
};

export const useIstioMesh = create<IstioMeshState>()((set) => {
  const fetchMesh = async (): Promise<void> => {
    try {
      set({ isLoading: true, error: null });

      const response = await fetch(import.meta.env.VITE_ISTIO_MESH_URL, {
        method: "GET",
      });
      if (!response.ok) throw new Error(`Response status ${response.status}`);

      const data: MeshGraph = await response.json();
      set({ data, isLoading: false });
    } catch (e) {
      set({ data: null, error: e as Error, isLoading: false });
    }
  };

  fetchMesh();

  return {
    isLoading: true,
    data: null,
    error: null,
    fetch: fetchMesh,
  };
});
