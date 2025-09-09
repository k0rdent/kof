import {
  Card,
  CardContent,
  CardHeader,
  CardTitle
} from "@/components/generated/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow
} from "@/components/generated/ui/table";
import { JSX } from "react";
import { useNavigate } from "react-router-dom";
import CustomizedTableHead from "@/components/pages/collectorPage/components/collector-list/CollectorTableHead";
import HealthSummary from "@/components/shared/HealthSummary";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { K8sObject } from "@/models/k8sObject";
import { DashboardData } from "@/components/pages/dashboards/DashboardTypes";

const DashboardList = <Items extends K8sObjectSet<Item>, Item extends K8sObject>({
  store,
  icon: Icon,
  tableCols
}: DashboardData<Items, Item>): JSX.Element => {
  const { items } = store();
  const navigate = useNavigate();

  return (
    <Card className="w-full gap-3">
      <CardHeader>
        <CardTitle>
          <div className="flex gap-4 items-center w-full h-full">
            <Icon className="w-5 h-5" />
            <HealthSummary
              totalCount={items?.length ?? 0}
              healthyCount={items?.healthyCount ?? 0}
              unhealthyCount={items?.unhealthyCount ?? 0}
            />
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table className="w-full table-fixed">
          <TableHeader>
            <TableRow className="font-bold">
              {tableCols?.map((th) => (
                <CustomizedTableHead
                  key={th.head.text}
                  text={th.head.text}
                  width={th.head.width}
                />
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {items?.objects.map((item) => (
              <TableRow
                onClick={() => navigate(item.name)}
                key={`${item.namespace}-${item.name}`}
                className="cursor-pointer"
              >
                {tableCols?.map((table) => (
                  <TableCell className="font-medium" key={table.head.text}>
                    {table.valueFn(item)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default DashboardList;
