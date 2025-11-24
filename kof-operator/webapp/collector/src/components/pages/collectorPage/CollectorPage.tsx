import { JSX, useEffect } from "react";
import { Separator } from "@/components/generated/ui/separator";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import CollectorsList from "./components/collector-list/CollectorsList";
import CollectorsPageHeader from "./CollectorsPageHeader";

const CollectorMetricsPage = (): JSX.Element => {
  const { data, selectedCluster, setSelectedCluster, setSelectedPod } =
    useCollectorMetricsState();

  useEffect(() => {
    if (data && !selectedCluster && data.clusters.length > 0) {
      setSelectedCluster(data.clusters[0].name);
      return;
    }

    if (selectedCluster && selectedCluster.customResource.length > 0) {
      setSelectedPod(selectedCluster.customResource[0].name);
    }
  }, [data, selectedCluster, setSelectedCluster, setSelectedPod]);

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <CollectorsPageHeader />
      <Separator />
      <CollectorsList />
    </div>
  );
};

export default CollectorMetricsPage;
