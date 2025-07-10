import { JSX, useState } from "react";
import useSWR from "swr";
import { Separator } from "../../ui/separator";
import SelectItems from "./components/Select";
import CollectorContent from "./components/CollectorContent";
import { Loader } from "lucide-react";
import { Cluster, CollectorMetrics, Pod } from "./models";

const CollectorMetricsPage = (): JSX.Element => {
  const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null);
  const [selectedCollector, setSelectedCollector] = useState<Pod | null>(null);

  const { data, isLoading } = useSWR(
    import.meta.env.VITE_COLLECTOR_METRICS_URL,
    fetcher
  );

  if (isLoading)
    return (
      <div className="flex w-full h-screen justify-center items-center">
        <Loader className="animate-spin w-8 h-8"></Loader>
      </div>
    );

  if (!data)
    return (
      <div className="flex w-full h-screen justify-center items-center">
        You don't have any clusters
      </div>
    );

  const collectorMetrics: CollectorMetrics = new CollectorMetrics(
    data.clusters
  );

  const onClusterSelect = (clusterName: string): void => {
    setSelectedCluster(collectorMetrics.getCluster(clusterName));
  };

  const onCollectorSelect = (podName: string): void => {
    if (selectedCluster) {
      setSelectedCollector(selectedCluster.getPod(podName));
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
            items={collectorMetrics.clusterNames}
            callbackFn={onClusterSelect}
            disabled={isLoading}
            placeholder="Select a cluster"
          ></SelectItems>
          <SelectItems
            items={selectedCluster?.podNames ?? []}
            callbackFn={onCollectorSelect}
            disabled={isLoading || !selectedCluster}
            placeholder="Select a collector"
          ></SelectItems>
        </div>
      </header>

      <Separator className="mb-3"></Separator>
      {selectedCollector && (
        <CollectorContent collector={selectedCollector}></CollectorContent>
      )}
    </div>
  );
};

const fetcher = (url: string) => fetch(url).then((res) => res.json());

export default CollectorMetricsPage;