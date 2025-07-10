import { JSX } from "react";
import { Pod } from "../models";
import { TabsContent } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { formatNumber } from "./CollectorOverviewTab";
import { Separator } from "@/components/ui/separator";
import StatRow from "@/components/shared/StatRow";
import { METRICS } from "@/constants/metrics.constants";

const CollectorExporterTabContent = ({
  collector,
}: {
  collector: Pod;
}): JSX.Element => {
  const queueSize = collector.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_SIZE);
  const queueCapacity = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY
  );
  const logsTotal = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_SENT_LOG_RECORDS
  );
  const metricsTotal = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS
  );
  const failedLogsTotal = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_SEND_FAILED_LOG_RECORDS
  );
  const failedMetricsTotal = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_SEND_FAILED_METRIC_POINTS
  );

  return (
    <TabsContent
      value="exporter"
      className="grid gap-6 md:grid-cols-2 lg:grid-cols-3"
    >
      <QueueCard queueCapacity={queueCapacity} queueSize={queueSize} />
      <SentRecordsCard logsTotal={logsTotal} metricsTotal={metricsTotal} />
      <FailedRecordsCard
        failedLogsTotal={failedLogsTotal}
        failedMetricsTotal={failedMetricsTotal}
      />
    </TabsContent>
  );
};

const QueueCard = ({
  queueSize,
  queueCapacity,
}: {
  queueSize: number;
  queueCapacity: number;
}): JSX.Element => {
  const queueUtilization = (queueSize / queueCapacity) * 100;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">Queue Status</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow text="Capacity" value={queueCapacity} />
        <StatRow text="Current Size" value={queueSize} />
        <StatRow
          text="Utilization"
          value={`${queueUtilization.toFixed(1)}%`}
          valueStyles="text-sm"
          containerStyle="mb-2"
        />
        <Progress value={queueUtilization} />
      </CardContent>
    </Card>
  );
};

const SentRecordsCard = ({
  logsTotal,
  metricsTotal,
}: {
  logsTotal: number;
  metricsTotal: number;
}): JSX.Element => {
  const formattedLogsCount = formatNumber(logsTotal);
  const formattedMetricsCount = formatNumber(metricsTotal);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Sent Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow text="Log Records" value={formattedLogsCount} />
        <StatRow text="Metric Points" value={formattedMetricsCount} />
        <Separator />
        <div className="text-xs text-muted-foreground">
          Total records successfully exported
        </div>
      </CardContent>
    </Card>
  );
};

const FailedRecordsCard = ({
  failedLogsTotal,
  failedMetricsTotal,
}: {
  failedLogsTotal: number;
  failedMetricsTotal: number;
}): JSX.Element => {
  const formattedFailedLogs = formatNumber(failedLogsTotal);
  const formattedFailedMetrics = formatNumber(failedMetricsTotal);

  const logsValueStyle =
    failedLogsTotal > 0 ? "text-red-600" : "text-green-600";
  const metricsValueStyle =
    failedMetricsTotal > 0 ? "text-red-600" : "text-green-600";

  return (
    <Card>
      <CardHeader>
        <CardTitle>Failed Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow
          text="Failed Log Records"
          valueStyles={logsValueStyle}
          value={formattedFailedLogs}
        />
        <StatRow
          text="Failed Metric Points"
          valueStyles={metricsValueStyle}
          value={formattedFailedMetrics}
        />
        <Separator />
        <div className="text-xs text-muted-foreground">
          Records that failed to export
        </div>
      </CardContent>
    </Card>
  );
};

export default CollectorExporterTabContent;
