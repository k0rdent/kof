import { JSX } from "react";
import VictoriaNameTableRowCell from "./VictoriaNameTableRowCell";
import { Pod } from "../../collectorPage/models";
import { TableCell, TableRow } from "@/components/generated/ui/table";
import { useNavigate } from "react-router-dom";
import HealthBadge from "@/components/shared/HealthBadge";
import UtilizationCell from "../../collectorPage/components/collector-list/UtilizationCell";
import { METRICS, VICTORIA_METRICS } from "@/constants/metrics.constants";
import MetricTrendCell from "../../collectorPage/components/collector-list/MetricTrendCell";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";

const VictoriaTableRow = ({ pod }: { pod: Pod }): JSX.Element => {
  const navigate = useNavigate();
  const { metricsHistory } = useVictoriaMetricsState();

  return (
    <TableRow
      className="cursor-pointer"
      onClick={() => navigate(`/victoria/${pod.clusterName}/${pod.name}`)}
    >
      <VictoriaNameTableRowCell name={pod.name} />
      <TableCell className="font-medium">
        <HealthBadge isHealthy={pod.isHealthy} />
      </TableCell>
      <UtilizationCell
        usageMetric={METRICS.CONTAINER_RESOURCE_CPU_USAGE.name}
        limitMetric={METRICS.CONTAINER_RESOURCE_CPU_LIMIT.name}
        pod={pod}
      />
      <UtilizationCell
        usageMetric={METRICS.CONTAINER_RESOURCE_MEMORY_USAGE.name}
        limitMetric={METRICS.CONTAINER_RESOURCE_MEMORY_LIMIT.name}
        pod={pod}
      />
      <MetricTrendCell
        metric={VICTORIA_METRICS.VM_HTTP_REQUESTS_ALL_TOTAL.name}
        pod={pod}
        metricsHistory={metricsHistory}
      />
    </TableRow>
  );
};

export default VictoriaTableRow;
