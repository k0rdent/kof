import { TabsContent } from "@/components/generated/ui/tabs";
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import { JSX, useMemo } from "react";
import { ClusterCondition } from "../../models";
import { Card, CardContent, CardHeader } from "@/components/generated/ui/card";
import { CircleAlert, CircleCheckBig } from "lucide-react";
import { Badge } from "@/components/generated/ui/badge";

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
      <ConditionHeader />
      {sortedConditions.map((c) => (
        <ConditionRow key={c.type} condition={c} />
      ))}
    </TabsContent>
  );
};

export default ClusterDeploymentStatusTab;

const ConditionRow = ({ condition }: ConditionRowProps): JSX.Element => {
  const isHealthy: boolean = condition.isHealthy;

  return (
    <Card className="gap-2">
      <CardHeader className="flex flex-row justify-between items-center">
        <div className="flex gap-4">
          {isHealthy ? (
            <CircleCheckBig className="text-green-600 w-5 h-5" />
          ) : (
            <CircleAlert className="text-red-600 w-5 h-5" />
          )}
          <span className="font-medium">{condition.type}</span>
          {isHealthy ? (
            <Badge
              variant="default"
              className="bg-green-600 text-primary-foreground"
            >
              Ready
            </Badge>
          ) : (
            <Badge variant="destructive">Failed</Badge>
          )}
        </div>
        <div className="text-muted-foreground text-sm">
          {condition.lastTransitionTimeDate.toLocaleString()}
        </div>
      </CardHeader>

      <CardContent>
        {condition.reason && (
          <div className="flex gap-2">
            <span>Reason: </span>
            <span>{condition.reason}</span>
          </div>
        )}
        {condition.message && (
          <div className="flex gap-2">
            <span>Message: </span>
            <span>{condition.message}</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
};

const ConditionHeader = (): JSX.Element => {
  const { selectedCluster } = useClusterDeploymentsProvider();

  return (
    <div className="flex gap-2 text-sm font-medium">
      <div>{selectedCluster?.totalStatusCount} conditions</div>
      <span>•</span>
      <div className="text-green-600">
        {selectedCluster?.healthyStatusCount} healthy
      </div>
      <span>•</span>
      <div className="text-red-600">
        {selectedCluster?.unhealthyStatusCount} unhealthy
      </div>
    </div>
  );
};

interface ConditionRowProps {
  condition: ClusterCondition;
}
