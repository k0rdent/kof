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
import { Workflow } from "lucide-react";
import { JSX } from "react";
import { useNavigate } from "react-router-dom";
import { formatTime } from "@/utils/formatter";
import { useMultiClusterServiceProvider } from "@/providers/MultiClusterServicesProvider";
import CustomizedTableHead from "@/components/pages/collectorPage/components/collector-list/CollectorTableHead";
import HealthBadge from "@/components/shared/HealthBadge";
import HealthSummary from "@/components/shared/HealthSummary";

const MultiClusterServicesList = (): JSX.Element => {
  return (
    <Card className="w-full gap-3">
      <ListHeader />
      <ListContent />
    </Card>
  );
};

export default MultiClusterServicesList;

const ListHeader = (): JSX.Element => {
  const { items: multiClusterServices } = useMultiClusterServiceProvider();
  return (
    <CardHeader>
      <CardTitle>
        <div className="flex gap-4 items-center w-full h-full">
          <Workflow className="w-5 h-5" />
          <HealthSummary
            totalCount={multiClusterServices?.length ?? 0}
            healthyCount={multiClusterServices?.healthyCount ?? 0}
            unhealthyCount={multiClusterServices?.unhealthyCount ?? 0}
            totalText={"Total"}
          />
        </div>
      </CardTitle>
    </CardHeader>
  );
};

const ListContent = (): JSX.Element => {
  const { items: services } = useMultiClusterServiceProvider();
  const navigate = useNavigate();

  return (
    <CardContent>
      <Table className="w-full table-fixed">
        <TableHeader>
          <TableRow className="font-bold">
            <CustomizedTableHead text={"Name"} width={200} />
            <CustomizedTableHead text={"Status"} width={110} />
            <CustomizedTableHead text={"Services Ready"} width={150} />
            <CustomizedTableHead text={"Cluster Ready"} width={150} />
            <CustomizedTableHead text={"Age"} width={120} />
          </TableRow>
        </TableHeader>
        <TableBody>
          {services?.objects.map((service) => (
            <TableRow
              onClick={() => navigate(service.name)}
              key={`${service.namespace}-${service.name}`}
              className="cursor-pointer"
            >
              <TableCell className="font-medium truncate">
                {service.name}
              </TableCell>
              <TableCell className="font-medium">
                <HealthBadge isHealthy={service.isHealthy} />
              </TableCell>
              <TableCell className="font-medium">
                {service.getCondition("ServicesInReadyState")?.message}
              </TableCell>
              <TableCell className="font-medium">
                {service.getCondition("ClusterInReadyState")?.message}
              </TableCell>
              <TableCell className="font-medium">
                {formatTime(service.ageInSeconds)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </CardContent>
  );
};
