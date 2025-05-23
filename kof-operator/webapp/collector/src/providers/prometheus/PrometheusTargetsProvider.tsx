import {
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import PrometheusTargetsContext from "@/providers/prometheus/PrometheusContext";
import { Cluster, PrometheusTargets } from "@/models/PrometheusTarget";
import { toast } from "sonner";

export type FilterFunction = (data: Cluster[]) => Cluster[];

interface FilterEntry {
  id: string;
  name: string;
  fn: FilterFunction;
}

const PrometheusTargetProvider = ({ children }: { children: ReactNode }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [data, setData] = useState<PrometheusTargets | null>(null);
  const [filters, setFilters] = useState<FilterEntry[]>([]);
  const fetchInProgress = useRef(false);

  const fetchPrometheusTargets = async () => {
    try {
      if (loading || fetchInProgress.current || data) {
        return;
      }

      fetchInProgress.current = true;
      setLoading(true);
      setError(null);

      const response = await fetch("http://localhost:9090/api/targets", {
        method: "GET",
      });

      if (!response.ok) {
        throw new Error(`HTTP error! Status: ${response.status}`);
      }

      const result: PrometheusTargets = (await response.json()) ?? null;
      setData(result);
    } catch (err: any) {
      setError(err.message);
      toast.error("Failed to fetch prometheus targets", {
        description: err.message,
      });
    } finally {
      setLoading(false);
      fetchInProgress.current = false;
    }
  };

  useEffect(() => {
    if (!data && !loading && !fetchInProgress.current) {
      fetchPrometheusTargets();
    }
  }, []);

  const filteredData = useMemo(() => {
    if (!data) return null;

    let result = [...data.clusters];

    filters.forEach((filter) => {
      result = filter.fn(result);
    });

    return { clusters: result };
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
