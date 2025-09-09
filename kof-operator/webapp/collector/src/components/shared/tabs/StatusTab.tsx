import { JSX, useMemo } from "react";
import { Condition } from "@/models/ObjectMeta";
import HealthSummary from "@/components/shared/HealthSummary";
import ConditionCard from "@/components/shared/ConditionCard";

interface StatusTabProps {
  conditions: Condition[];
}

const StatusTab = ({
  conditions,
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
    <>
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
    </>
  );
};

export default StatusTab;
