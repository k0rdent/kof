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
import { Layers } from "lucide-react";
import { JSX } from "react";
import { useNavigate } from "react-router-dom";
import { formatTime } from "@/utils/formatter";
import { useClusterSummariesProvider } from "@/providers/cluster_summaries/ClusterDeploymentsProvider";
import CustomizedTableHead from "@/components/pages/collectorPage/components/collector-list/CollectorTableHead";
import HealthBadge from "@/components/shared/HealthBadge";
import HealthSummary from "@/components/shared/HealthSummary";

const ClusterSummariesList = (): JSX.Element => {
  return (
    <Card className="w-full gap-3">
      <ListHeader />
      <ListContent />
    </Card>
  );
};

export default ClusterSummariesList;

const ListHeader = (): JSX.Element => {
  const { data: summaries } = useClusterSummariesProvider();
  return (
    <CardHeader>
      <CardTitle>
        <div className="flex gap-4 items-center w-full h-full">
          <Layers className="w-5 h-5" />
          <HealthSummary
            totalCount={summaries?.length ?? 0}
            healthyCount={summaries?.healthyCount ?? 0}
            unhealthyCount={summaries?.unhealthyCount ?? 0}
            totalText={"Total"}
          />
        </div>
      </CardTitle>
    </CardHeader>
  );
};

const ListContent = (): JSX.Element => {
  const { data: summaries } = useClusterSummariesProvider();
  const navigate = useNavigate();

  return (
    <CardContent>
      <Table className="w-full table-fixed">
        <TableHeader>
          <TableRow className="font-bold">
            <CustomizedTableHead text={"Namespace"} width={110} />
            <CustomizedTableHead text={"Name"} width={200} />
            <CustomizedTableHead text={"Status"} width={110} />
            <CustomizedTableHead text={"Age"} width={120} />
          </TableRow>
        </TableHeader>
        <TableBody>
          {summaries?.clusterSummariesArray.map((cluster) => (
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
                {formatTime(cluster.ageInSeconds)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </CardContent>
  );
};
