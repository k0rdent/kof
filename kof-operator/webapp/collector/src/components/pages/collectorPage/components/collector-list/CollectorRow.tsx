import { TableCell, TableRow } from "@/components/generated/ui/table";

import { JSX } from "react";
import { useNavigate } from "react-router-dom";
import { Pod } from "../../models";
import { METRICS } from "@/constants/metrics.constants";
import CollectorNameCell from "./CollectorNameCell";
import HealthBadge from "@/components/shared/HealthBadge";
import UtilizationCell from "./UtilizationCell";
import MetricTrendCell from "./MetricTrendCell";

const CollectorRow = ({ pod }: { pod: Pod }): JSX.Element => {
  const navigate = useNavigate();

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
        usageMetric={METRICS.OTELCOL_CONTAINER_RESOURCE_CPU_USAGE}
        limitMetric={METRICS.OTELCOL_CONTAINER_RESOURCE_CPU_LIMIT}
        pod={pod}
      />
      <UtilizationCell
        usageMetric={METRICS.OTELCOL_CONTAINER_RESOURCE_MEMORY_USAGE}
        limitMetric={METRICS.OTELCOL_CONTAINER_RESOURCE_MEMORY_LIMIT}
        pod={pod}
      />
      <UtilizationCell
        usageMetric={METRICS.OTELCOL_EXPORTER_QUEUE_SIZE}
        limitMetric={METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY}
        pod={pod}
      />
      <MetricTrendCell
        pod={pod}
        metric={METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS}
      />
      <MetricTrendCell
        pod={pod}
        metric={METRICS.OTELCOL_EXPORTER_SENT_LOG_RECORDS}
      />
    </TableRow>
  );
};

export default CollectorRow;
