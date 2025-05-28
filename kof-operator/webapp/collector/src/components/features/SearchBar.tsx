import { ChangeEvent, JSX, useEffect, useState } from "react";
import { Input } from "../ui/input";
import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";
import { Cluster } from "@/models/Cluster";
import { FilterFunction } from "@/providers/prometheus/PrometheusTargetsProvider";
import { Target } from "@/models/PrometheusTarget";

const SearchBar = (): JSX.Element => {
  const [filterId, setFilterId] = useState<string | null>(null);
  const { loading, addFilter, removeFilter } = usePrometheusTarget();

  useEffect(() => {
    return () => {
      if (filterId) {
        removeFilter(filterId);
      }
    };
  }, [removeFilter, filterId]);

  const handleInputs = (e: ChangeEvent<HTMLInputElement>) => {
    const value: string = e.currentTarget.value;

    if (filterId) {
      removeFilter(filterId);
    }

    if (value) {
      const id = addFilter("search", SearchFilter(value));
      setFilterId(id);
    } else {
      setFilterId(null);
    }
  };

  return (
    <div className="w-full min-w-[250px] max-w-[350px]">
      <Input
        disabled={loading}
        onChange={handleInputs}
        type="text"
        placeholder="Search by endpoints, labels or scrape pool"
      ></Input>
    </div>
  );
};

export default SearchBar;

const SearchFilter = (value: string): FilterFunction => {
  return (clusters: Cluster[]) => {
    if (!value) return clusters;

    const targetFilterFn = (target: Target): boolean => {
      const includeInLabels = Object.values(target.labels).some((val) =>
        val.includes(value)
      );

      const includeInDiscoveredLabels = Object.values(
        target.discoveredLabels
      ).some((val) => val.includes(value));

      return (
        target.scrapeUrl.includes(value) ||
        target.scrapePool.includes(value) ||
        includeInLabels ||
        includeInDiscoveredLabels
      );
    };

    return clusters
      .map((cluster) => cluster.filterTargets(targetFilterFn))
      .filter((cluster) => cluster.hasNodes);
  };
};
