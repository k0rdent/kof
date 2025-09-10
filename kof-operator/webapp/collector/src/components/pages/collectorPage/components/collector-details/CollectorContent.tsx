import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/generated/ui/tabs";
import { JSX, useEffect } from "react";
import CollectorProcessorTab from "./CollectorProcessorTab";
import CollectorReceiverTab from "./CollectorReceiverTab";
import CollectorExporterTab from "./CollectorExporterTab";
import CollectorOverviewTab from "./CollectorOverviewTab";
import CollectorProcessTab from "./CollectorProcessTab";
import UnhealthyAlert from "@/components/shared/UnhealthyAlert";
import ContentHeader from "@/components/shared/ContentHeader";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { useParams } from "react-router-dom";
import CollectorsPageHeader from "../../CollectorsPageHeader";
import { Separator } from "@/components/generated/ui/separator";
import { Loader, MoveLeft } from "lucide-react";
import { Button } from "@/components/generated/ui/button";
import { useNavigate } from "react-router-dom";
import RawJsonTab from "@/components/shared/tabs/RawJsonTab";

const CollectorContent = (): JSX.Element => {
  const {
    setSelectedPod: setSelectedCollector,
    setSelectedCluster,
    selectedPod: selectedCollector,
    isLoading,
    data,
  } = useCollectorMetricsState();

  const navigate = useNavigate();

  const { cluster, collector } = useParams();
  useEffect(() => {
    if (!isLoading && cluster && collector) {
      setSelectedCluster(cluster);
      setSelectedCollector(collector);
    }
  }, [cluster, collector, setSelectedCluster, setSelectedCollector, isLoading]);

  if (isLoading && !data) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <CollectorsPageHeader />
        <Separator />
        <div className="flex w-full h-full justify-center items-center">
          <Loader className="animate-spin w-8 h-8"></Loader>
        </div>
      </div>
    );
  }

  if (!selectedCollector) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <CollectorsPageHeader />
        <Separator />
        <div className="flex flex-col w-full h-full justify-center items-center space-y-4">
          <span className="font-bold text-2xl">Collector not found</span>
          <Button
            variant="outline"
            className="cursor-pointer"
            onClick={() => {
              navigate("/collectors");
            }}
          >
            <MoveLeft />
            <span>Back to Table</span>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <CollectorsPageHeader />
      <Separator />
      <ContentHeader
        tableURL={"/collectors"}
        title={"Collector"}
        pod={selectedCollector}
        state={useCollectorMetricsState}
      />

      <UnhealthyAlert pod={selectedCollector} />
      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList className="flex w-full">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="exporter">Exporter</TabsTrigger>
          <TabsTrigger value="processor">Processor</TabsTrigger>
          <TabsTrigger value="receiver">Receiver</TabsTrigger>
          <TabsTrigger value="process">Process</TabsTrigger>
          <TabsTrigger value="raw_json">Raw Metrics</TabsTrigger>
        </TabsList>
        <CollectorOverviewTab collector={selectedCollector} />
        <CollectorExporterTab />
        <CollectorProcessorTab />
        <CollectorReceiverTab />
        <CollectorProcessTab />
        <TabsContent value="raw_json">
          <RawJsonTab object={selectedCollector.getMetrics()} />
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default CollectorContent;
