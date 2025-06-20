import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";
import { JSX, useState } from "react";
import SearchBar from "./SearchBar";
import HealthSelector from "./HealthSelector";
import PopoverSelector, {
  PopoverClusterFilter,
  PopoverNodeFilter,
} from "./PopoverSelector";

const TargetFilter = (): JSX.Element => {
  const [selectedClusters, setSelectedClusters] = useState<string[]>([]);

  const { data } = usePrometheusTarget();

  return (
    <div className="w-full lg:flex-row lg:items-center p-6 flex gap-5 flex-col items-start">
      <PopoverSelector
        id="cluster"
        labelContent="Clusters: "
        placeholderText="Search clusters..."
        popoverButtonText="Select clusters..."
        noValuesText="No clusters found."
        dataToDisplay={Array.from(data?.clusters ?? [])}
        filterFn={PopoverClusterFilter}
        onSelectionChange={setSelectedClusters}
      ></PopoverSelector>
      <PopoverSelector
        id="node"
        labelContent="Nodes: "
        placeholderText="Search nodes..."
        popoverButtonText="Select nodes..."
        noValuesText="No nodes found."
        filterFn={PopoverNodeFilter}
        dataToDisplay={
          data?.clusters
            .filter(
              (cluster) =>
                selectedClusters.length === 0 ||
                selectedClusters.includes(cluster.name)
            )
            .flatMap((c) => c.nodes) ?? []
        }
      ></PopoverSelector>
      <SearchBar></SearchBar>
      <HealthSelector></HealthSelector>
    </div>
  );
};

export default TargetFilter;
