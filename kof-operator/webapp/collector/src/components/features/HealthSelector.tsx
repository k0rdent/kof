import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";
import { JSX, useEffect, useState } from "react";
import { Checkbox } from "../ui/checkbox";
import { Badge } from "../ui/badge";
import { Cluster } from "@/models/PrometheusTarget";
import { FilterFunction } from "@/providers/prometheus/PrometheusTargetsProvider";
import { Label } from "../ui/label";
import { getTargets } from "@/utils/cluster";

type State = "up" | "down" | "unknown";

interface SelectorItemProps {
  name: string;
  color: string;
  state: State;
  count: number;
}

const SelectorItems: SelectorItemProps[] = [
  {
    name: "Unknown",
    color: "bg-amber-300 text-black",
    state: "unknown",
    count: 0,
  },
  {
    name: "Down",
    color: "bg-red-500",
    state: "down",
    count: 0,
  },
  {
    name: "Up",
    color: "bg-green-500",
    state: "up",
    count: 0,
  },
];

const HealthSelector = (): JSX.Element => {
  const [filterId, setFilterId] = useState<string | null>(null);
  const [states, setStates] = useState<State[]>([]);
  const [selectorItems, setSelectorItems] =
    useState<SelectorItemProps[]>(SelectorItems);
  const { data, loading, addFilter, removeFilter } = usePrometheusTarget()!;

  useEffect(() => {
    if (loading) return;

    const targets = getTargets(data?.clusters ?? []);
    const updatedItems = selectorItems.map((item) => ({
      ...item,
      count: targets.filter((target) => target.health === item.state).length,
    }));

    setSelectorItems(updatedItems);
  }, [data]);

  const onCheckboxClick = (selectorId: string) => {
    const selectedState = SelectorItems.find(
      (item) => item.name === selectorId
    )?.state;

    if (!selectedState) return;
    let newStates: State[];

    if (states.includes(selectedState)) {
      newStates = states.filter((state) => state !== selectedState);
    } else {
      newStates = [...states, selectedState];
    }

    setStates(newStates);

    if (filterId) {
      removeFilter(filterId);
    }

    if (newStates.length > 0) {
      const newFilterId = addFilter("health_filter", HealthFilter(newStates));
      setFilterId(newFilterId);
    } else {
      setFilterId(null);
    }
  };

  return (
    <div className="flex gap-3 w-full justify-end">
      {selectorItems.map((selector) => (
        <Label className="flex gap-1 items-center h-fit cursor-pointer">
          <Checkbox
            className="cursor-pointer"
            onClick={() => onCheckboxClick(selector.name)}
            disabled={loading}
          ></Checkbox>
          <Badge className={`${selector.color}`}>
            {selector.count} {selector.name}
          </Badge>
        </Label>
      ))}
    </div>
  );
};

export default HealthSelector;

const HealthFilter = (states: State[]): FilterFunction => {
  return (data: Cluster[]) => {
    if (states.length == 0) return data;

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
                            (target) => states.includes(target.health as State)
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
