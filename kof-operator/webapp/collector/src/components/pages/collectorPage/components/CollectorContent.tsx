import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { JSX } from "react";
import { Pod } from "../models";

import CollectorProcessorTab from "./CollectorProcessorTab";
import CollectorReceiverTab from "./CollectorReceiverTab";
import CollectorExporterTab from "./CollectorExporterTab";
import CollectorOverviewTab from "./CollectorOverviewTab";
import CollectorProcessTab from "./CollectorProcessTab";
import UnhealthyAlert from "./UnhealthyAlert";
import CollectorContentHeader from "./CollectorContentHeader";

export interface CollectorProps {
  collector: Pod;
}

const CollectorContent = ({ collector }: CollectorProps): JSX.Element => {
  return (
    <div className="space-y-5">
      <CollectorContentHeader collector={collector} />

      {!collector.isHealthy && <UnhealthyAlert collector={collector} />}

      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList className="flex w-full">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="exporter">Exporter</TabsTrigger>
          <TabsTrigger value="processor">Processor</TabsTrigger>
          <TabsTrigger value="receiver">Receiver</TabsTrigger>
          <TabsTrigger value="process">Process</TabsTrigger>
        </TabsList>
        <CollectorOverviewTab collector={collector}></CollectorOverviewTab>
        <CollectorExporterTab collector={collector}></CollectorExporterTab>
        <CollectorProcessorTab collector={collector}></CollectorProcessorTab>
        <CollectorReceiverTab collector={collector}></CollectorReceiverTab>
        <CollectorProcessTab collector={collector}></CollectorProcessTab>
      </Tabs>
    </div>
  );
};

export default CollectorContent;
