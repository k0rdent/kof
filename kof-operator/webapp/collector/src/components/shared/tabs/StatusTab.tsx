import { JSX, useMemo } from "react";
import { Condition } from "@/models/ObjectMeta";
import { TabsContent } from "@/components/generated/ui/tabs";
import HealthSummary from "@/components/shared/HealthSummary";
import ConditionCard from "@/components/shared/ConditionCard";

interface StatusTabProps {
  conditions: Condition[];
  tabValue?: string;
}

const StatusTab = ({
  conditions,
  tabValue = "status",
}: StatusTabProps): JSX.Element => {
  const sortedConditions: Condition[] = useMemo(() => {
    return (
      conditions.sort((a, b) => Number(!b.isHealthy) - Number(!a.isHealthy)) ??
      []
    );
  }, [conditions]);

  const healthyCount: number = useMemo(() => {
    return conditions.filter((c) => c.isHealthy).length;
  }, [conditions]);

  const unhealthyCount: number = conditions.length - healthyCount;

  return (
    <TabsContent value={tabValue} className="flex flex-col gap-5">
      <HealthSummary
        className="text-sm font-medium"
        totalCount={conditions.length}
        healthyCount={healthyCount}
        unhealthyCount={unhealthyCount}
        totalText={"total"}
      />
      {sortedConditions.map((c) => (
        <ConditionCard key={c.name} condition={c} />
      ))}
    </TabsContent>
  );
};

export default StatusTab;
