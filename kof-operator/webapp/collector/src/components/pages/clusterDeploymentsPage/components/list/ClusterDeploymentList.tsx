import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from "@/components/generated/ui/table";
import { Server } from "lucide-react";
import { JSX } from "react";
import CustomizedTableHead from "../../../collectorPage/components/collector-list/CollectorTableHead";
import HealthBadge from "@/components/shared/HealthBadge";
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import { useNavigate } from "react-router-dom";
import { capitalizeFirstLetter, formatTime } from "@/utils/formatter";

const ClusterDeploymentsList = (): JSX.Element => {
  return (
    <Card className="w-full gap-3">
      <ListHeader />
      <ListContent />
    </Card>
  );
};

export default ClusterDeploymentsList;

const ListHeader = (): JSX.Element => {
  const { data: clusters } = useClusterDeploymentsProvider();
  return (
    <CardHeader>
      <CardTitle>
        <div className="flex gap-4 items-center w-full h-full">
          <Server className="w-5 h-5" />
          <div className="flex gap-3 w-full items-center">
            <span>{clusters?.length} Total</span>
            <span>•</span>
            <span className="text-green-600">
              {clusters?.healthyCount} Healthy
            </span>
            <span>•</span>
            <span className="text-red-600">
              {clusters?.unhealthyCount} Unhealthy
            </span>
          </div>
        </div>
      </CardTitle>
    </CardHeader>
  );
};

const ListContent = (): JSX.Element => {
  const { data: clusters } = useClusterDeploymentsProvider();
  const navigate = useNavigate();

  return (
    <CardContent>
      <Table className="w-full table-fixed">
        <TableHeader>
          <TableRow className="font-bold">
            <CustomizedTableHead text={"Namespace"} width={110} />
            <CustomizedTableHead text={"Name"} width={200} />
            <CustomizedTableHead text={"Status"} width={110} />
            <CustomizedTableHead text={"Role"} width={100} />
            <CustomizedTableHead text={"Template"} width={180} />
            <CustomizedTableHead text={"Age"} width={120} />
          </TableRow>
        </TableHeader>
        <TableBody>
          {clusters?.deployments.map((cluster) => (
            <TableRow
              onClick={() => navigate(cluster.name)}
              key={`${cluster.namespace}-${cluster.name}`}
              className="cursor-pointer"
            >
              <TableCell className="font-medium">{cluster.namespace}</TableCell>
              <TableCell className="font-medium truncate">
                {cluster.name}
              </TableCell>
              <TableCell className="font-medium">
                <HealthBadge isHealthy={cluster.isHealthy} />
              </TableCell>
              <TableCell className="font-medium">
                {capitalizeFirstLetter(cluster.role ?? "N/A")}
              </TableCell>
              <TableCell className="font-medium">
                {cluster.spec.template}
              </TableCell>
              <TableCell className="font-medium">
                {formatTime(cluster.ageInSeconds)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </CardContent>
  );
};
