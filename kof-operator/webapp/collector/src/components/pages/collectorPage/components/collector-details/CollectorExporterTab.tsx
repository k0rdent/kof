import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/generated/ui/card";
import { Progress } from "@/components/generated/ui/progress";
import { Separator } from "@/components/generated/ui/separator";
import StatRowWithTrend from "@/components/shared/StatRowWithTrend";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";
import { getMetricTrendData } from "@/utils/metrics";
import StatRow from "@/components/shared/StatRow";

const CollectorExporterTabContent = (): JSX.Element => {
  return (
    <TabsContent
      value="exporter"
      className="grid gap-6 md:grid-cols-2 lg:grid-cols-3"
    >
      <QueueCard />
      <SentRecordsCard />
      <FailedRecordsCard />
    </TabsContent>
  );
};

const QueueCard = (): JSX.Element => {
  const { selectedCollector: col } = useCollectorMetricsState();

  if (!col) {
    return <></>;
  }

  const queueCapacity = col.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY);
  const queueSize = col.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_SIZE);
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

const SentRecordsCard = (): JSX.Element => {
  const { metricsHistory, selectedCollector: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const { metricValue: logValue, metricTrend: logTrend } = getMetricTrendData(
    METRICS.OTELCOL_EXPORTER_SENT_LOG_RECORDS,
    metricsHistory,
    col,
    timePeriod
  );

  const { metricValue, metricTrend } = getMetricTrendData(
    METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS,
    metricsHistory,
    col,
    timePeriod
  );

  const formattedLogsCount = formatNumber(logValue);
  const formattedMetricsCount = formatNumber(metricValue);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Sent Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRowWithTrend
          text="Log Records"
          value={formattedLogsCount}
          trend={logTrend}
          isPositiveTrend={true}
        />
        <StatRowWithTrend
          text="Metric Points"
          value={formattedMetricsCount}
          trend={metricTrend}
          isPositiveTrend={true}
        />
        <Separator />
        <div className="text-xs text-muted-foreground">
          Total records successfully exported
        </div>
      </CardContent>
    </Card>
  );
};

const FailedRecordsCard = (): JSX.Element => {
  const { metricsHistory, selectedCollector: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const { metricValue: logValue, metricTrend: logTrend } = getMetricTrendData(
    METRICS.OTELCOL_EXPORTER_SEND_FAILED_LOG_RECORDS,
    metricsHistory,
    col,
    timePeriod
  );

  const { metricValue, metricTrend } = getMetricTrendData(
    METRICS.OTELCOL_EXPORTER_SEND_FAILED_METRIC_POINTS,
    metricsHistory,
    col,
    timePeriod
  );

  const formattedFailedLogs = formatNumber(logValue);
  const formattedFailedMetrics = formatNumber(metricValue);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Failed Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRowWithTrend
          text="Failed Log Records"
          value={formattedFailedLogs}
          trend={logTrend}
          isPositiveTrend={false}
        />
        <StatRowWithTrend
          text="Failed Metric Points"
          value={formattedFailedMetrics}
          trend={metricTrend}
          isPositiveTrend={false}
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
