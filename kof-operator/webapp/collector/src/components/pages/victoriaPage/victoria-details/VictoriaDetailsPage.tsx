import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { JSX, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Separator } from "@/components/generated/ui/separator";
import { Loader, MoveLeft } from "lucide-react";
import { Button } from "@/components/generated/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/generated/ui/tabs";
import UnhealthyAlert from "@/components/shared/UnhealthyAlert";
import VictoriaPageHeader from "../VictoriaPageHeader";
import VictoriaOverviewTab from "./VictoriaOverviewTab";
import VictoriaSystemTab from "./VictoriaSystemTab";
import VictoriaGoRuntimeTab from "./VictoriaGoRuntimeTab";
import VictoriaNetworkTab from "./VictoriaNetworkTab";
import ContentHeader from "../../../shared/ContentHeader";
import { getVictoriaNameType, getVictoriaType } from "../utils";
import VictoriaMetricsInsertTab from "./VictoriaMetricsInsertTab";
import VictoriaMetricsSelectTab from "./VictoriaMetricsSelectTab";
import VictoriaLogsInsertTab from "./VictoriaLogsInsertTab";
import VictoriaMetricsStorageTab from "./VictoriaMetricsStorageTab";
import VictoriaLogsSelectTab from "./VictoriaLogsSelectTab";
import VictoriaLogsStorageTab from "./VictoriaLogsStorageTab";
import RawJsonTab from "@/components/shared/tabs/RawJsonTab";

const VictoriaDetailsPage = (): JSX.Element => {
  const { setSelectedPod, setSelectedCluster, selectedPod, isLoading, data } =
    useVictoriaMetricsState();

  const navigate = useNavigate();

  const { cluster, pod } = useParams();
  useEffect(() => {
    if (!isLoading && cluster && pod) {
      setSelectedCluster(cluster);
      setSelectedPod(pod);
    }
  }, [cluster, pod, setSelectedCluster, setSelectedPod, isLoading]);

  if (isLoading && !data) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <VictoriaPageHeader />
        <Separator />
        <div className="flex w-full h-full justify-center items-center">
          <Loader className="animate-spin w-8 h-8"></Loader>
        </div>
      </div>
    );
  }

  if (!selectedPod) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <VictoriaPageHeader />
        <Separator />
        <div className="flex flex-col w-full h-full justify-center items-center space-y-4">
          <span className="font-bold text-2xl">Victoria pods not found</span>
          <Button
            variant="outline"
            className="cursor-pointer"
            onClick={() => {
              navigate("/victoria");
            }}
          >
            <MoveLeft />
            <span>Back to Table</span>
          </Button>
        </div>
      </div>
    );
  }

  const podType = getVictoriaType(selectedPod.name);

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <VictoriaPageHeader />
      <Separator />
      <ContentHeader
        tableURL={"/victoria"}
        title={getVictoriaNameType(selectedPod.name)}
        pod={selectedPod}
        state={useVictoriaMetricsState}
      />

      <UnhealthyAlert pod={selectedPod} />
      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList className="flex w-full cursor-pointer">
          <TabsTrigger className="cursor-pointer" value="overview">
            Overview
          </TabsTrigger>
          <TabsTrigger className="cursor-pointer" value="system">
            System
          </TabsTrigger>
          <TabsTrigger className="cursor-pointer" value="network">
            Network
          </TabsTrigger>
          <TabsTrigger className="cursor-pointer" value="go_runtime">
            Go Runtime
          </TabsTrigger>

          {podType === "vminsert" && (
            <TabsTrigger className="cursor-pointer" value="vm_insert">
              VictoriaMetrics Insert
            </TabsTrigger>
          )}

          {podType === "vmselect" && (
            <TabsTrigger className="cursor-pointer" value="vm_select">
              VictoriaMetrics Select
            </TabsTrigger>
          )}

          {podType === "vmstorage" && (
            <TabsTrigger className="cursor-pointer" value="vm_storage">
              VictoriaMetrics Storage
            </TabsTrigger>
          )}

          {podType === "vlinsert" && (
            <TabsTrigger className="cursor-pointer" value="vl_insert">
              VictoriaLogs Insert
            </TabsTrigger>
          )}

          {podType === "vlselect" && (
            <TabsTrigger className="cursor-pointer" value="vl_select">
              VictoriaLogs Select
            </TabsTrigger>
          )}

          {podType === "vlstorage" && (
            <TabsTrigger className="cursor-pointer" value="vl_storage">
              VictoriaLogs Storage
            </TabsTrigger>
          )}

          <TabsTrigger value={"raw_json"}>Raw Metrics</TabsTrigger>
        </TabsList>

        <VictoriaOverviewTab />
        <VictoriaSystemTab />
        <VictoriaNetworkTab />
        <VictoriaGoRuntimeTab />
        <VictoriaMetricsInsertTab />
        <VictoriaMetricsSelectTab />
        <VictoriaMetricsStorageTab />
        <VictoriaLogsInsertTab />
        <VictoriaLogsSelectTab />
        <VictoriaLogsStorageTab />
        <RawJsonTab object={selectedPod.getMetrics()} />
      </Tabs>
    </div>
  );
};

export default VictoriaDetailsPage;
