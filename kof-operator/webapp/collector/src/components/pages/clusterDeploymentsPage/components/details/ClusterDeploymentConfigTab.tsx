import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";
import StatRow from "@/components/shared/StatRow";
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import { Cpu, FileSliders } from "lucide-react";
import { JSX } from "react";

const ClusterDeploymentConfigurationTab = (): JSX.Element => {
  return (
    <TabsContent value="configuration" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2">
        <InfrastructureCard />
        <ClusterConfigurationCard />
      </div>
    </TabsContent>
  );
};

export default ClusterDeploymentConfigurationTab;

const ClusterConfigurationCard = (): JSX.Element => {
  const { selectedCluster } = useClusterDeploymentsProvider();
  const rows: MetricRow[] = [
    { title: "Template", value: selectedCluster?.spec.template },
    { title: "Credential", value: selectedCluster?.spec.credential },
  ];

  return (
    <MetricsCard rows={rows} icon={FileSliders} title="Cluster Configuration" />
  );
};

const InfrastructureCard = (): JSX.Element => {
  const { selectedCluster } = useClusterDeploymentsProvider();
  const rows: MetricRow[] = [
    { title: "Cloud Provider", value: selectedCluster?.spec.provider },
    { title: "Region", value: selectedCluster?.spec.config.region },
    {
      title: "Control Plane",
      customRow: ({ title }) => {
        const nodesNumber = selectedCluster?.spec.config.controlPlaneNumber;
        const type = selectedCluster?.spec.config.controlPlane.instanceType;
        return <StatRow text={title} value={`${nodesNumber} x ${type}`} />;
      },
    },
    {
      title: "Workers",
      customRow: ({ title }) => {
        const nodesNumber = selectedCluster?.spec.config.workersNumber;
        const type = selectedCluster?.spec.config.worker.instanceType;
        return <StatRow text={title} value={`${nodesNumber} x ${type}`} />;
      },
    },
  ];

  return <MetricsCard rows={rows} icon={Cpu} title="Infrastructure" />;
};
