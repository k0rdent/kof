import { TableCell, TableRow } from "@/components/generated/ui/table";

import { JSX } from "react";
import { useNavigate } from "react-router-dom";
import { Pod } from "../../models";
import { METRICS } from "@/constants/metrics.constants";
import CollectorNameCell from "./CollectorNameCell";
import HealthBadge from "@/components/shared/HealthBadge";
import UtilizationCell from "./UtilizationCell";
import MetricTrendCell from "./MetricTrendCell";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";

const CollectorRow = ({ pod }: { pod: Pod }): JSX.Element => {
  const navigate = useNavigate();
  const { metricsHistory } = useCollectorMetricsState();

  return (
    <TableRow
      className="cursor-pointer"
      onClick={() => navigate(`/collectors/${pod.clusterName}/${pod.name}`)}
    >
      <CollectorNameCell name={pod.name} />
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
      <UtilizationCell
        usageMetric={METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.name}
        limitMetric={METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.name}
        pod={pod}
      />
      <MetricTrendCell
        pod={pod}
        metric={METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS.name}
        metricsHistory={metricsHistory}
      />
      <MetricTrendCell
        pod={pod}
        metric={METRICS.OTELCOL_EXPORTER_SENT_LOG_RECORDS.name}
        metricsHistory={metricsHistory}
      />
    </TableRow>
  );
};

export default CollectorRow;
