import { JSX, useEffect } from "react";
import { Separator } from "@/components/generated/ui/separator";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import CollectorsList from "./components/collector-list/CollectorsList";
import CollectorsPageHeader from "./CollectorsPageHeader";

const CollectorMetricsPage = (): JSX.Element => {
  const { data, selectedCluster, setSelectedCluster, setSelectedCollector } =
    useCollectorMetricsState();

  useEffect(() => {
    if (data && !selectedCluster) {
      setSelectedCluster(data.clusters[0].name);
      return;
    }

    if (selectedCluster && selectedCluster.pods.length > 0) {
      setSelectedCollector(selectedCluster.pods[0].name);
    }
  }, [data, selectedCluster, setSelectedCluster, setSelectedCollector]);

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <CollectorsPageHeader />
      <Separator />
      <CollectorsList />
    </div>
  );
};

export default CollectorMetricsPage;
