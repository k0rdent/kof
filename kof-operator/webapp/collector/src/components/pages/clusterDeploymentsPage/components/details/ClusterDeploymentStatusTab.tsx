import { TabsContent } from "@/components/generated/ui/tabs";
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import { JSX, useMemo } from "react";
import { ClusterCondition } from "../../models";
import ConditionCard from "@/components/shared/ConditionCard";
import HealthSummary from "@/components/shared/HealthSummary";

const ClusterDeploymentStatusTab = (): JSX.Element => {
  const { selectedCluster } = useClusterDeploymentsProvider();

  const sortedConditions: ClusterCondition[] = useMemo(() => {
    return (
      selectedCluster?.status.conditions.sort(
        (a, b) => Number(!b.isHealthy) - Number(!a.isHealthy)
      ) ?? []
    );
  }, [selectedCluster?.status.conditions]);

  return (
    <TabsContent value="status" className="flex flex-col gap-5">
      <HealthSummary
        totalCount={selectedCluster?.totalStatusCount ?? 0}
        healthyCount={selectedCluster?.healthyStatusCount ?? 0}
        unhealthyCount={selectedCluster?.unhealthyStatusCount ?? 0}
        totalText={"conditions"}
        className="text-sm font-medium"
      />
      {sortedConditions.map((c) => (
        <ConditionCard key={c.name} condition={c} />
      ))}
    </TabsContent>
  );
};

export default ClusterDeploymentStatusTab;
