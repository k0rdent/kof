import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import JsonView from "@uiw/react-json-view";
import {
  Clock,
  Database,
  FileText,
  LucideProps,
  Tag,
  Tags,
} from "lucide-react";
import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";

const ClusterDeploymentMetadataTab = (): JSX.Element => {
  const { selectedCluster: cluster } = useClusterDeploymentsProvider();

  return (
    <TabsContent value="metadata" className="max-w-full flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <BasicInfoCard />
        <TimelineCard />
      </div>

      <JsonMetricsCard title="Labels" icon={Tag} data={cluster?.labels ?? {}} />
      <JsonMetricsCard
        title="Cluster Annotations"
        icon={FileText}
        data={cluster?.spec.config.clusterAnnotations ?? {}}
      />
      <JsonMetricsCard
        title="Annotations"
        icon={Tags}
        data={cluster?.annotations ?? {}}
      />
    </TabsContent>
  );
};

export default ClusterDeploymentMetadataTab;

const BasicInfoCard = (): JSX.Element => {
  const { selectedCluster: cluster } = useClusterDeploymentsProvider();

  const rows: MetricRow[] = [
    { title: "Name", value: cluster?.name },
    { title: "Namespace", value: cluster?.namespace },
    { title: "Generation", value: String(cluster?.generation) },
  ];

  return (
    <MetricsCard rows={rows} icon={Database} title={"Basic Information"} />
  );
};

const TimelineCard = (): JSX.Element => {
  const { selectedCluster: cluster } = useClusterDeploymentsProvider();

  const rows: MetricRow[] = [
    { title: "Created", value: cluster?.creationTime.toLocaleString() },
    {
      title: "Deletion Started",
      value: cluster?.deletionTime?.toLocaleString() ?? "Not Deleted",
    },
  ];

  return <MetricsCard rows={rows} icon={Clock} title={"Timeline"} />;
};

interface JsonMetricsCardProps {
  title: string;
  icon: ForwardRefExoticComponent<
    Omit<LucideProps, "ref"> & RefAttributes<SVGSVGElement>
  >;
  data: Record<string, string>;
}

const JsonMetricsCard = ({
  title,
  icon,
  data,
}: JsonMetricsCardProps): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "",
      customRow: () => (
        <div className="flex flex-col gap-2 w-full">
          <JsonView
            value={data}
            displayDataTypes={false}
            className="w-full whitespace-normal break-words"
          />
        </div>
      ),
    },
  ];

  return <MetricsCard rows={rows} icon={icon} title={title} />;
};
