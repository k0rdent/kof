import { Button } from "@/components/generated/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/generated/ui/tabs";
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import { FileText, HardDrive, MoveLeft } from "lucide-react";
import { JSX, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import HealthBadge from "@/components/shared/HealthBadge";
import ClusterDeploymentStatusTab from "./ClusterDeploymentStatusTab";
import ClusterDeploymentConfigurationTab from "./ClusterDeploymentConfigTab";
import MetadataTab from "@/components/shared/tabs/MetadataTab";
import JsonViewCard from "@/components/shared/JsonViewCard";

const ClusterDeploymentDetails = (): JSX.Element => {
  const {
    data,
    setSelectedCluster,
    selectedCluster: cluster,
    isLoading,
  } = useClusterDeploymentsProvider();

  const navigate = useNavigate();
  const { clusterName } = useParams();

  useEffect(() => {
    if (!isLoading && data && clusterName) {
      setSelectedCluster(clusterName);
    }
  }, [clusterName, data, isLoading, setSelectedCluster]);

  if (!cluster) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <div className="flex flex-col w-full h-full justify-center items-center space-y-4">
          <span className="font-bold text-2xl">
            Cluster deployments not found
          </span>
          <Button
            variant="outline"
            className="cursor-pointer"
            onClick={() => {
              navigate("/cluster-deployments");
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
      <DetailsHeader />
      <Tabs defaultValue="status" className="space-y-6">
        <TabsList className="flex w-full">
          <TabsTrigger value="status">Status</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
          <TabsTrigger value="metadata">Metadata</TabsTrigger>
        </TabsList>
        <ClusterDeploymentStatusTab />
        <ClusterDeploymentConfigurationTab />
        <MetadataTab metadata={cluster.metadata}>
          <JsonViewCard
            title="Cluster Annotations"
            icon={FileText}
            data={cluster?.spec.config.clusterAnnotations ?? {}}
          />
        </MetadataTab>
      </Tabs>
    </div>
  );
};

export default ClusterDeploymentDetails;

const DetailsHeader = (): JSX.Element => {
  const { selectedCluster: cluster } = useClusterDeploymentsProvider();
  const navigate = useNavigate();

  return (
    <div className="space-y-6">
      <Button
        variant="outline"
        className="cursor-pointer"
        onClick={() => {
          navigate("/cluster-deployments");
        }}
      >
        <MoveLeft />
        <span>Back to Table</span>
      </Button>
      <div className="flex gap-4 items-center mb-2">
        <HardDrive />
        <span className="font-bold text-xl">{cluster?.name}</span>
        <HealthBadge isHealthy={cluster?.isHealthy ?? false} />
      </div>
    </div>
  );
};
