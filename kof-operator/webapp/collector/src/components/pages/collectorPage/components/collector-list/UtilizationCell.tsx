import { JSX } from "react";
import { Pod } from "../../models";
import { TableCell } from "@/components/generated/ui/table";
import { Progress } from "@/components/generated/ui/progress";

interface UtilizationCellProps {
  usageMetric: string;
  limitMetric: string;
  pod: Pod;
}

const UtilizationCell = ({
  pod,
  usageMetric,
  limitMetric,
}: UtilizationCellProps): JSX.Element => {
  const currentUsage: number = pod.getMetric(usageMetric)?.totalValue ?? 0;
  const currentLimit: number = pod.getMetric(limitMetric)?.totalValue ?? 0;

  const usagePercentage =
    currentLimit > 0 ? (currentUsage / currentLimit) * 100 : 0;
  const formattedUsagePercentage = usagePercentage.toFixed(1);

  return (
    <TableCell>
      <div className="flex gap-3 items-center">
        <span>{formattedUsagePercentage} %</span>
        <Progress value={usagePercentage}></Progress>
      </div>
    </TableCell>
  );
};

export default UtilizationCell;
