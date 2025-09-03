import { TabsContent } from "@/components/generated/ui/tabs";
import { JSX, useMemo } from "react";
import { useClusterSummariesProvider } from "@/providers/cluster_summaries/ClusterDeploymentsProvider";
import { HelmReleaseSummary } from "../models";
import ConditionCard from "@/components/shared/ConditionCard";
import HealthSummary from "@/components/shared/HealthSummary";

const ClusterSummaryHelmTab = (): JSX.Element => {
  const { selectedSummary: summary } = useClusterSummariesProvider();

  const sortedConditions: HelmReleaseSummary[] = useMemo(() => {
    return (
      summary?.status.helmReleaseSummaries.arr.sort(
        (a, b) => Number(!b.isHealthy) - Number(!a.isHealthy)
      ) ?? []
    );
  }, [summary?.status.helmReleaseSummaries.arr]);

  return (
    <TabsContent value="helm" className="flex flex-col gap-5">
      <HealthSummary
        totalCount={summary?.status.helmReleaseSummaries.count ?? 0}
        healthyCount={summary?.status.helmReleaseSummaries.healthyCount ?? 0}
        unhealthyCount={
          summary?.status.helmReleaseSummaries.unhealthyCount ?? 0
        }
        totalText={"helms"}
        className="text-sm font-medium"
      />
      {sortedConditions.map((c) => (
        <ConditionCard key={c.hash} condition={c}>
          <div className="flex gap-2">
            <span>Release Namespace: </span>
            <span>{c.releaseNamespace}</span>
          </div>
        </ConditionCard>
      ))}
    </TabsContent>
  );
};

export default ClusterSummaryHelmTab;
