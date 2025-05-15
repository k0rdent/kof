import { ChangeEvent, JSX, useEffect, useState } from "react";
import { Input } from "../ui/input";
import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";
import { Cluster } from "@/models/PrometheusTarget";
import { FilterFunction } from "@/providers/prometheus/PrometheusTargetsProvider";

const SearchBar = (): JSX.Element => {
  const [filterId, setFilterId] = useState<string | null>(null);
  const { loading, addFilter, removeFilter } = usePrometheusTarget()!;

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
      >
      </Input>
    </div>
  );
};

export default SearchBar;

const SearchFilter = (value: string): FilterFunction => {
  return (data: Cluster[]) => {
    if (!value) return data;

    return data
      .map((cluster) => {
        return {
          ...cluster,
          nodes: cluster.nodes
            .map((node) => {
              return {
                ...node,
                pods: node.pods
                  .map((pod) => {
                    return {
                      ...pod,
                      response: {
                        ...pod.response,
                        data: {
                          ...pod.response.data,
                          activeTargets: pod.response.data.activeTargets.filter(
                            (target) => {
                              const includeInLabelsValue = Array.from(
                                target.labels.values()
                              ).some((labelValue) =>
                                labelValue.includes(value)
                              );

                              const includeInLabelsKey = Array.from(
                                target.labels.keys()
                              ).some((key) => key.includes(value));

                              return (
                                target.scrapeUrl.includes(value) ||
                                target.scrapePool.includes(value) ||
                                includeInLabelsValue ||
                                includeInLabelsKey
                              );
                            }
                          ),
                        },
                      },
                    };
                  })
                  .filter((pod) => pod.response.data.activeTargets.length > 0),
              };
            })
            .filter((node) => node.pods.length > 0),
        };
      })
      .filter((cluster) => cluster.nodes.length > 0);
  };
};
