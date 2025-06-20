import {
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import PrometheusTargetsContext from "@/providers/prometheus/PrometheusContext";
import { toast } from "sonner";
import { Cluster, ClustersData } from "@/models/Cluster";
import { PrometheusTargetsManager } from "@/models/PrometheusTargetsManager";

export type FilterFunction = (data: Cluster[]) => Cluster[];

interface FilterEntry {
  id: string;
  name: string;
  fn: FilterFunction;
}

const PrometheusTargetProvider = ({ children }: { children: ReactNode }) => {
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<Error | null>(null);
  const [data, setData] = useState<PrometheusTargetsManager | null>(null);
  const [filters, setFilters] = useState<FilterEntry[]>([]);
  const fetchInProgress = useRef(false);

  const fetchPrometheusTargets = useCallback(async () => {
    try {
      if (fetchInProgress.current) {
        return;
      }

      fetchInProgress.current = true;
      setLoading(true);
      setError(null);

      const response = await fetch(import.meta.env.VITE_TARGET_URL, {
        method: "GET",
      });

      if (!response.ok) {
        throw new Error(`HTTP error! Status: ${response.status}`);
      }

      const data: ClustersData = await response.json();
      setData(
        new PrometheusTargetsManager({
          clusters: data.clusters,
        })
      );
      setError(null);
    } catch (err) {
      if (err instanceof Error) {
        setError(err);
        toast.error("Failed to fetch prometheus targets", {
          description: err.message,
        });
      }
    } finally {
      setLoading(false);
      fetchInProgress.current = false;
    }
  }, []);

  useEffect(() => {
    fetchPrometheusTargets();
  }, [fetchPrometheusTargets]);

  const filteredData = useMemo(() => {
    if (!data) return null;

    let result = data.clusters;

    filters.forEach((filter) => {
      result = filter.fn(result);
    });

    return result;
  }, [data, filters]);

  const addFilter = useCallback((name: string, filterFn: FilterFunction) => {
    const id = `filter_${name}_${Date.now()}`;
    setFilters((prev) => [...prev, { id, name, fn: filterFn }]);
    return id;
  }, []);

  const removeFilter = useCallback((id: string) => {
    setFilters((prev) => prev.filter((filter) => filter.id !== id));
  }, []);

  const clearFilters = useCallback(() => {
    setFilters([]);
  }, []);

  return (
    <PrometheusTargetsContext.Provider
      value={{
        loading,
        error,
        data,
        filteredData,
        addFilter,
        removeFilter,
        clearFilters,
        fetchPrometheusTargets,
      }}
    >
      {children}
    </PrometheusTargetsContext.Provider>
  );
};

export default PrometheusTargetProvider;
