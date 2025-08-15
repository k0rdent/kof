import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { CheckCircle, TriangleAlert } from "lucide-react";

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
  const rows: MetricCardRow[] = [
    {
      title: "Log Records",
      metricName: METRICS.OTELCOL_RECEIVER_ACCEPTED_LOG_RECORDS.name,
      hint: METRICS.OTELCOL_RECEIVER_ACCEPTED_LOG_RECORDS.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Metric Points",
      metricName: METRICS.OTELCOL_RECEIVER_ACCEPTED_METRIC_POINTS.name,
      hint: METRICS.OTELCOL_RECEIVER_ACCEPTED_METRIC_POINTS.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={CheckCircle}
      state={useCollectorMetricsState}
      title="Successfully Received Records"
    />
  );
};

const RefusedRecordsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Log Records",
      metricName: METRICS.OTELCOL_RECEIVER_REFUSED_LOG_RECORDS.name,
      hint: METRICS.OTELCOL_RECEIVER_REFUSED_LOG_RECORDS.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Metric Points",
      metricName: METRICS.OTELCOL_RECEIVER_REFUSED_METRIC_POINTS.name,
      hint: METRICS.OTELCOL_RECEIVER_REFUSED_METRIC_POINTS.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={TriangleAlert}
      state={useCollectorMetricsState}
      title="Rejected or Failed Records"
    />
  );
};
