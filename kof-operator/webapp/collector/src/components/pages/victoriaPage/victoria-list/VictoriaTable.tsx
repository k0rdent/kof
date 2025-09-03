import { JSX } from "react";
import {
  Table,
  TableBody,
  TableHeader,
  TableRow,
} from "@/components/generated/ui/table";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import { Layers } from "lucide-react";
import { Cluster } from "../../collectorPage/models";
import CustomizedTableHead from "../../collectorPage/components/collector-list/CollectorTableHead";
import VictoriaTableRow from "./VictoriaTableRow";
import HealthSummary from "@/components/shared/HealthSummary";

const VictoriaTable = ({ cluster }: { cluster: Cluster }): JSX.Element => {
  return (
    <Card className="w-full h-full gap-3">
      <CardHeader>
        <CardTitle>
          <div className="flex gap-4 items-center w-full h-full">
            <Layers className="w-5 h-5s"></Layers>
            <div className="flex gap-1 flex-col">
              <h1 className="text-lg font-bold">{cluster.name}</h1>
              <HealthSummary
                totalCount={cluster.pods.length}
                healthyCount={cluster.healthyPodCount}
                unhealthyCount={cluster.unhealthyPodCount}
                totalText={"pods"}
                className="text-sm font-medium"
              />
            </div>
          </div>
        </CardTitle>
      </CardHeader>

      <CardContent>
        <Table className="w-full table-fixed">
          <TableHeader>
            <TableRow className="font-bold">
              <CustomizedTableHead text={"Pod Name"} width={200} />
              <CustomizedTableHead text={"Status"} width={110} />
              <CustomizedTableHead text={"CPU %"} width={150} />
              <CustomizedTableHead text={"Memory %"} width={150} />
              <CustomizedTableHead text={"HTTP Requests"} width={150} />
            </TableRow>
          </TableHeader>
          <TableBody>
            {cluster.pods.map((pod) => (
              <VictoriaTableRow key={pod.name} pod={pod} />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default VictoriaTable;
