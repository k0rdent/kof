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
import { ServerCog } from "lucide-react";
import { JSX } from "react";
import { useNavigate } from "react-router-dom";
import { formatTime } from "@/utils/formatter";
import { useStateManagementProvidersProvider } from "@/providers/StateManagementProvidersProvider";
import CustomizedTableHead from "@/components/pages/collectorPage/components/collector-list/CollectorTableHead";
import HealthBadge from "@/components/shared/HealthBadge";
import HealthSummary from "@/components/shared/HealthSummary";

const StateManagementProviderList = (): JSX.Element => {
  return (
    <Card className="w-full gap-3">
      <ListHeader />
      <ListContent />
    </Card>
  );
};

export default StateManagementProviderList;

const ListHeader = (): JSX.Element => {
  const { items: providers } = useStateManagementProvidersProvider();
  return (
    <CardHeader>
      <CardTitle>
        <div className="flex gap-4 items-center w-full h-full">
          <ServerCog className="w-5 h-5" />
          <HealthSummary
            totalCount={providers?.length ?? 0}
            healthyCount={providers?.healthyCount ?? 0}
            unhealthyCount={providers?.unhealthyCount ?? 0}
          />
        </div>
      </CardTitle>
    </CardHeader>
  );
};

const ListContent = (): JSX.Element => {
  const { items: providers } = useStateManagementProvidersProvider();
  const navigate = useNavigate();

  return (
    <CardContent>
      <Table className="w-full table-fixed">
        <TableHeader>
          <TableRow className="font-bold">
            <CustomizedTableHead text={"Name"} width={200} />
            <CustomizedTableHead text={"Status"} width={110} />
            <CustomizedTableHead text={"Age"} width={120} />
          </TableRow>
        </TableHeader>
        <TableBody>
          {providers?.objects.map((provider) => (
            <TableRow
              onClick={() => navigate(provider.name)}
              key={`${provider.namespace}-${provider.name}`}
              className="cursor-pointer"
            >
              <TableCell className="font-medium truncate">
                {provider.name}
              </TableCell>
              <TableCell className="font-medium">
                <HealthBadge isHealthy={provider.isHealthy} />
              </TableCell>
              <TableCell className="font-medium">
                {formatTime(provider.ageInSeconds)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </CardContent>
  );
};
