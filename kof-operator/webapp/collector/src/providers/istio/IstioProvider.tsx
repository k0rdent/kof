import { create } from "zustand";

export const useIstio = create<IstioProviderState>()((set) => {
  const fetchMetrics = async (): Promise<void> => {
    try {
      set({ isLoading: true, error: null });
      const response = await fetch(import.meta.env.VITE_ISTIO_SECRETS_URL, {
        method: "GET",
      });

      if (!response.ok) {
        throw new Error(`Response status ${response.status}`);
      }

      const resp: IstioSecretsResponse = await response.json();
      set({
        data: resp,
        isLoading: false,

      });
    } catch (e) {
      set({ data: null, error: e as Error, isLoading: false });
    }
  };


  fetchMetrics();

  return {
    isLoading: true,
    data: null,
    error: null,
    fetch: fetchMetrics,
  };
});

export type IstioSecretsResponse = {
  secrets: Secret[];
};

type Secret = {
  name: string;
  namespace: string;
  syncStatus: string;
  clusterName: string;
};

type IstioProviderState = {
  isLoading: boolean;
  data: IstioSecretsResponse | null;
  error: Error | null;
  fetch: () => Promise<void>;
};
