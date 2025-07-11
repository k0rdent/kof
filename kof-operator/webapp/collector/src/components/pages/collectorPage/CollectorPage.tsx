import { JSX, useEffect } from "react";
import { Separator } from "@/components/generated/ui/separator";
import SelectItems from "./components/Select";
import CollectorContent from "./components/CollectorContent";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";

const CollectorMetricsPage = (): JSX.Element => {
  const {
    isLoading,
    data,
    selectedCluster,
    selectedCollector,
    setSelectedCluster,
    setSelectedCollector,
  } = useCollectorMetricsState();

  useEffect(() => {
    if (data && !selectedCluster) {
      setSelectedCluster(data.clusters[0].name);
      return;
    }

    if (selectedCluster) {
      setSelectedCollector(selectedCluster.pods[0].name);
    }
  }, [data, selectedCluster, setSelectedCluster, setSelectedCollector]);

  const onClusterSelect = (clusterName: string): void => {
    if (data) {
      setSelectedCluster(clusterName);
    }
  };

  const onCollectorSelect = (podName: string): void => {
    if (selectedCluster) {
      setSelectedCollector(podName);
    }
  };

  return (
    <div className="flex flex-col w-full h-full p-5">
      <header className="flex justify-between">
        <h1 className="font-bold text-3xl pb-3">
          OpenTelemetry Collectors Metrics
        </h1>
        <div className="flex gap-2">
          <SelectItems
            items={data?.clusterNames ?? []}
            callbackFn={onClusterSelect}
            disabled={isLoading}
            placeholder="Select a cluster"
            value={selectedCluster?.name}
          ></SelectItems>
          <SelectItems
            items={selectedCluster?.podNames ?? []}
            callbackFn={onCollectorSelect}
            disabled={isLoading || !selectedCluster}
            placeholder="Select a collector"
            value={selectedCollector?.name}
          ></SelectItems>
        </div>
      </header>

      <Separator className="mb-3" />

      <CollectorContent />
    </div>
  );
};

export default CollectorMetricsPage;
