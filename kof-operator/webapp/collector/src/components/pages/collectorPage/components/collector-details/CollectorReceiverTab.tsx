import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/generated/ui/card";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";
import { getMetricTrendData } from "@/utils/metrics";
import StatRowWithTrend from "@/components/shared/StatRowWithTrend";

const CollectorReceiverTab = (): JSX.Element => {
  return (
    <TabsContent value="receiver">
      <div className="grid gap-6 md:grid-cols-2">
        <AcceptedRecordsCard />
        <RefusedRecordsCard />
      </div>
    </TabsContent>
  );
};

export default CollectorReceiverTab;

const AcceptedRecordsCard = (): JSX.Element => {
  const { metricsHistory, selectedCollector: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const { metricValue: logValue, metricTrend: logTrend } = getMetricTrendData(
    METRICS.OTELCOL_RECEIVER_ACCEPTED_LOG_RECORDS,
    metricsHistory,
    col,
    timePeriod
  );

  const { metricValue, metricTrend } = getMetricTrendData(
    METRICS.OTELCOL_RECEIVER_ACCEPTED_METRIC_POINTS,
    metricsHistory,
    col,
    timePeriod
  );

  const formattedLogRecordsReceived = formatNumber(logValue);
  const formattedMetricRecordsReceived = formatNumber(metricValue);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Successfully Received Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRowWithTrend
          text="Log Records"
          value={formattedLogRecordsReceived}
          trend={logTrend}
          isPositiveTrend={true}
        />
        <StatRowWithTrend
          text="Metric Points"
          value={formattedMetricRecordsReceived}
          trend={metricTrend}
          isPositiveTrend={true}
        />
      </CardContent>
    </Card>
  );
};

const RefusedRecordsCard = (): JSX.Element => {
  const { metricsHistory, selectedCollector: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const { metricValue: logValue, metricTrend: logTrend } = getMetricTrendData(
    METRICS.OTELCOL_RECEIVER_REFUSED_LOG_RECORDS,
    metricsHistory,
    col,
    timePeriod
  );

  const { metricValue, metricTrend } = getMetricTrendData(
    METRICS.OTELCOL_RECEIVER_REFUSED_METRIC_POINTS,
    metricsHistory,
    col,
    timePeriod
  );

  const formattedRefusedLogRecords = formatNumber(logValue);
  const formattedRefusedMetricRecords = formatNumber(metricValue);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Rejected or Failed Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRowWithTrend
          text="Log Records"
          value={formattedRefusedLogRecords}
          trend={logTrend}
          isPositiveTrend={false}
        />
        <StatRowWithTrend
          text="Metric Points"
          value={formattedRefusedMetricRecords}
          trend={metricTrend}
          isPositiveTrend={false}
        />
      </CardContent>
    </Card>
  );
};
