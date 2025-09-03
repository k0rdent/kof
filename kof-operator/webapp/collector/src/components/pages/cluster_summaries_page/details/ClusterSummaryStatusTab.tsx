import { TabsContent } from "@/components/generated/ui/tabs";
import { JSX, useMemo } from "react";
import { useClusterSummariesProvider } from "@/providers/cluster_summaries/ClusterDeploymentsProvider";
import { FeatureSummary } from "../models";
import ConditionCard from "@/components/shared/ConditionCard";
import HealthSummary from "@/components/shared/HealthSummary";

const ClusterSummaryStatusTab = (): JSX.Element => {
  const { selectedSummary: summary } = useClusterSummariesProvider();

  const sortedConditions: FeatureSummary[] = useMemo(() => {
    return (
      summary?.status.featureSummaries.arr.sort(
        (a, b) => Number(!b.isHealthy) - Number(!a.isHealthy)
      ) ?? []
    );
  }, [summary?.status.featureSummaries]);

  return (
    <TabsContent value="status" className="flex flex-col gap-5">
      <HealthSummary
        totalCount={summary?.status.featureSummaries.count ?? 0}
        healthyCount={summary?.status.featureSummaries.healthyCount ?? 0}
        unhealthyCount={summary?.status.featureSummaries.unhealthyCount ?? 0}
        totalText={"total"}
        className="text-sm font-medium"
      />
      {sortedConditions.map((c) => (
        <ConditionCard key={c.hash} condition={c} />
      ))}
    </TabsContent>
  );
};

export default ClusterSummaryStatusTab;
