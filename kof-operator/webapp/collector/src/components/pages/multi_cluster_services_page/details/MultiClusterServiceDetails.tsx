import { Button } from "@/components/generated/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/generated/ui/tabs";
import { Layers2, MoveLeft } from "lucide-react";
import { JSX, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMultiClusterServiceProvider } from "@/providers/MultiClusterServicesProvider";
import MetadataTab from "@/components/shared/tabs/MetadataTab";
import RawJsonTab from "@/components/shared/tabs/RawJsonTab";
import StatusTab from "@/components/shared/tabs/StatusTab";
import DetailsHeader from "@/components/shared/DetailsHeader";

const MultiClusterServiceDetails = (): JSX.Element => {
  const {
    isLoading,
    items: services,
    selectedItem: service,
    selectItem,
  } = useMultiClusterServiceProvider();

  const navigate = useNavigate();
  const { serviceName } = useParams();

  useEffect(() => {
    if (!isLoading && services && serviceName) {
      selectItem(serviceName);
    }
  }, [serviceName, services, isLoading, selectItem]);

  if (!service) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <div className="flex flex-col w-full h-full justify-center items-center space-y-4">
          <span className="font-bold text-2xl">
            Multi cluster service not found
          </span>
          <Button
            variant="outline"
            className="cursor-pointer"
            onClick={() => {
              navigate("/cluster-summaries");
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
    <div className="flex flex-col gap-5">
      <DetailsHeader
        icon={Layers2}
        title={service.name}
        isHealthy={service.isHealthy}
      />
      <Tabs defaultValue="status" className="space-y-6">
        <TabsList className="flex w-full">
          <TabsTrigger value="status">Status</TabsTrigger>
          <TabsTrigger value="metadata">Metadata</TabsTrigger>
          <TabsTrigger value="raw_json">Raw Json</TabsTrigger>
        </TabsList>
        <StatusTab conditions={service.status.conditions} />
        <MetadataTab metadata={service.metadata} />
        <RawJsonTab depthLevel={4} object={service.raw} />
      </Tabs>
    </div>
  );
};

export default MultiClusterServiceDetails;
