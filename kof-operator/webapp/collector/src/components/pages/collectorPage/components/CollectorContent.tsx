import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { JSX } from "react";

import CollectorProcessorTab from "./CollectorProcessorTab";
import CollectorReceiverTab from "./CollectorReceiverTab";
import CollectorExporterTab from "./CollectorExporterTab";
import CollectorOverviewTab from "./CollectorOverviewTab";
import CollectorProcessTab from "./CollectorProcessTab";
import UnhealthyAlert from "./UnhealthyAlert";
import CollectorContentHeader from "./CollectorContentHeader";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { Loader } from "lucide-react";
import { Button } from "@/components/ui/button";

const CollectorContent = (): JSX.Element => {
  const { isLoading, data, error, selectedCollector, fetch } = useCollectorMetricsState();

  if (isLoading) {
    return (
      <div className="flex w-full justify-center items-center mt-[15%]">
        <Loader className="animate-spin w-8 h-8"></Loader>
      </div>
    );
  }

  if (!isLoading && !data) {
    return (
      <div className="flex w-full h-screen justify-center items-center">
        You don't have any clusters
      </div>
    );
  }

  if (!isLoading && error) {
    return (
      <div className="flex flex-col justify-center items-center mt-[15%]">
        <span className="mb-3">
          Failed to fetch collectors metrics. Click "Reload" button to try
          again.
        </span>
        <Button className="cursor-pointer" onClick={fetch}>
          Reload
        </Button>
      </div>
    );
  }

  if (!selectedCollector) {
    return (
        <div className="flex w-full h-screen justify-center items-center">
        Please select the collector
      </div>
    )
  }

  return (
    <div className="space-y-5">
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
        <CollectorOverviewTab collector={selectedCollector}></CollectorOverviewTab>
        <CollectorExporterTab collector={selectedCollector}></CollectorExporterTab>
        <CollectorProcessorTab collector={selectedCollector}></CollectorProcessorTab>
        <CollectorReceiverTab collector={selectedCollector}></CollectorReceiverTab>
        <CollectorProcessTab collector={selectedCollector}></CollectorProcessTab>
      </Tabs>
    </div>
  );
};

export default CollectorContent;
