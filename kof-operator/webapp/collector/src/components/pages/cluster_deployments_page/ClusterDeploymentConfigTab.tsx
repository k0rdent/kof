import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";
import { Cpu, FileSliders } from "lucide-react";
import { JSX } from "react";
import { ClusterDeployment } from "./models";
import StatRow from "@/components/shared/StatRow";

interface ConfigurationTabProps {
  clusterDeployment: ClusterDeployment;
}

const ClusterDeploymentConfigurationTab = ({
  clusterDeployment
}: ConfigurationTabProps): JSX.Element => {
  return (
    <div className="grid gap-6 md:grid-cols-2">
      <InfrastructureCard clusterDeployment={clusterDeployment} />
      <ClusterConfigurationCard clusterDeployment={clusterDeployment} />
    </div>
  );
};

export default ClusterDeploymentConfigurationTab;

const ClusterConfigurationCard = ({
  clusterDeployment: cluster
}: ConfigurationTabProps): JSX.Element => {
  const rows: MetricRow[] = [
    { title: "Template", value: cluster.spec.template },
    { title: "Credential", value: cluster.spec.credential }
  ];

  return <MetricsCard rows={rows} icon={FileSliders} title="Cluster Configuration" />;
};

const InfrastructureCard = ({ clusterDeployment: cluster }: ConfigurationTabProps) => {
  const rows: MetricRow[] = [{ title: "Cloud Provider", value: cluster.spec.provider }];

  if (cluster.spec.config.region) {
    rows.push({ title: "Region", value: cluster.spec.config.region });
  }

  const addNodeRow = (
    rows: MetricRow[],
    title: string,
    count: number,
    type?: string
  ) => {
    if (!type) return;
    rows.push({
      title,
      customRow: ({ title }) => <StatRow text={title} value={`${count} x ${type}`} />
    });
  };

  addNodeRow(
    rows,
    "Control Plane",
    cluster.spec.config.controlPlaneNumber ?? 0,
    cluster.spec.config.controlPlane?.instanceType
  );

  addNodeRow(
    rows,
    "Workers",
    cluster.spec.config.workersNumber ?? 0,
    cluster.spec.config.worker?.instanceType
  );

  return <MetricsCard rows={rows} icon={Cpu} title="Infrastructure" />;
};
