import { Tabs, TabsList, TabsTrigger } from "@/components/generated/ui/tabs";
import { JSX, useEffect } from "react";
import CollectorProcessorTab from "./CollectorProcessorTab";
import CollectorReceiverTab from "./CollectorReceiverTab";
import CollectorExporterTab from "./CollectorExporterTab";
import CollectorOverviewTab from "./CollectorOverviewTab";
import CollectorProcessTab from "./CollectorProcessTab";
import UnhealthyAlert from "./UnhealthyAlert";
import CollectorContentHeader from "./CollectorContentHeader";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { useParams } from "react-router-dom";
import CollectorsPageHeader from "../../CollectorsPageHeader";
import { Separator } from "@/components/generated/ui/separator";

const CollectorContent = (): JSX.Element => {
  const { setSelectedCollector, setSelectedCluster, selectedCollector } =
    useCollectorMetricsState();

  const { cluster, collector } = useParams();
  useEffect(() => {
    if (cluster && collector) {
      setSelectedCluster(cluster);
      setSelectedCollector(collector);
    }
  }, [cluster, collector, setSelectedCluster, setSelectedCollector]);

  if (!selectedCollector) {
    return (
      <div className="flex w-full h-screen justify-center items-center">
        Please select the collector
      </div>
    );
  }

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <CollectorsPageHeader />
      <Separator />
      <CollectorContentHeader />

      <UnhealthyAlert />
      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList className="flex w-full">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="exporter">Exporter</TabsTrigger>
          <TabsTrigger value="processor">Processor</TabsTrigger>
          <TabsTrigger value="receiver">Receiver</TabsTrigger>
          <TabsTrigger value="process">Process</TabsTrigger>
        </TabsList>
        <CollectorOverviewTab
          collector={selectedCollector}
        ></CollectorOverviewTab>
        <CollectorExporterTab />
        <CollectorProcessorTab />
        <CollectorReceiverTab />
        <CollectorProcessTab />
      </Tabs>
    </div>
  );
};

export default CollectorContent;
