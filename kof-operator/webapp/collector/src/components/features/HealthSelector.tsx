import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";
import { JSX, useEffect, useState } from "react";
import { Checkbox } from "@/components/generated/ui/checkbox";
import { Badge } from "@/components/generated/ui/badge";
import { Cluster } from "@/models/Cluster";
import { FilterFunction } from "@/providers/prometheus/PrometheusTargetsProvider";
import { Label } from "@/components/generated/ui/label";
import { Target } from "@/models/PrometheusTarget";

export type State = "up" | "down" | "unknown";

interface TargetHealth {
  name: string;
  color: string;
  state: State;
  role: string;
  count: number;
}

const TargetHealth: TargetHealth[] = [
  {
    name: "Unknown",
    color: "bg-amber-300 text-black",
    state: "unknown",
    role: "unknown-checkbox",
    count: 0,
  },
  {
    name: "Down",
    color: "bg-red-500",
    state: "down",
    role: "down-checkbox",
    count: 0,
  },
  {
    name: "Up",
    color: "bg-green-500",
    state: "up",
    role: "up-checkbox",
    count: 0,
  },
];

const HealthSelector = (): JSX.Element => {
  const [filterId, setFilterId] = useState<string | null>(null);
  const [states, setStates] = useState<State[]>([]);
  const [selectorItems, setSelectorItems] =
    useState<TargetHealth[]>(TargetHealth);
  const { data, loading, addFilter, removeFilter } = usePrometheusTarget();

  useEffect(() => {
    if (!data) return;

    const targets: Target[] = data.targets;
    const updatedItems = TargetHealth.map((item) => ({
      ...item,
      count: targets.filter((target) => target.health === item.state).length,
    }));

    setSelectorItems(updatedItems);
  }, [data]);

  const onCheckboxClick = (selectorId: string) => {
    const selectedState = TargetHealth.find(
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
        <Label
          key={selector.state}
          className="flex gap-1 items-center h-fit cursor-pointer"
        >
          <Checkbox
            className="cursor-pointer"
            role={selector.role}
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

export const HealthFilter = (states: State[]): FilterFunction => {
  return (data: Cluster[]) => {
    if (states.length == 0) return data;

    const filterFn = (target: Target): boolean => {
      return states.includes(target.health as State);
    };

    return data
      .map((cluster) => cluster.filterTargets(filterFn))
      .filter((cluster) => cluster.hasNodes);
  };
};
