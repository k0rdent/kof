import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Progress } from "@/components/generated/ui/progress";
import { Separator } from "@/components/generated/ui/separator";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { Clock, Send, TriangleAlert } from "lucide-react";

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
  const { selectedPod: pod } = useCollectorMetricsState();

  if (!pod) {
    return <></>;
  }

  const rows: MetricCardRow[] = [
    {
      title: "Capacity",
      metricName: METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.name,
      hint: METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.hint,
    },
    {
      title: "Current Size",
      metricName: METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.name,
      hint: METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.hint,
    },
    {
      title: "Utilization",
      metricFetchFn: (pod) => {
        const cap = pod.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.name);
        const size = pod.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.name);
        return (size / cap) * 100;
      },
      metricFormat: (val) => `${val.toFixed(1)}%`,
      hint: "Percentage of the exporter queue currently in use"
    },
    {
      title: "Utilization Bar",
      metricFetchFn: (pod) => {
        const cap = pod.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.name);
        const size = pod.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.name);
        return (size / cap) * 100;
      },
      customRow: ({ rawValue, title }) => (
        <Progress key={title} value={rawValue} />
      ),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Clock}
      state={useCollectorMetricsState}
      title={"Queue Status"}
    />
  );
};

const SentRecordsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Log Records",
      metricName: METRICS.OTELCOL_EXPORTER_SENT_LOG_RECORDS.name,
      hint: METRICS.OTELCOL_EXPORTER_SENT_LOG_RECORDS.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Metric Points",
      metricName: METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS.name,
      hint: METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },

    {
      title: "Sent Records Description",
      customRow: ({ title }) => {
        return (
          <div key={title} className="space-y-2">
            <Separator />
            <div className="text-xs text-muted-foreground">
              Total records successfully exported
            </div>
          </div>
        );
      },
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Send}
      state={useCollectorMetricsState}
      title={"Sent Records"}
    />
  );
};

const FailedRecordsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Failed Log Records",
      metricName: METRICS.OTELCOL_EXPORTER_SEND_FAILED_LOG_RECORDS.name,
      hint: METRICS.OTELCOL_EXPORTER_SEND_FAILED_LOG_RECORDS.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Failed Metric Points",
      metricName: METRICS.OTELCOL_EXPORTER_SEND_FAILED_METRIC_POINTS.name,
      hint: METRICS.OTELCOL_EXPORTER_SEND_FAILED_METRIC_POINTS.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },

    {
      title: "Failed Records Description",
      customRow: ({ title }) => {
        return (
          <div key={title} className="space-y-2">
            <Separator />
            <div className="text-xs text-muted-foreground">
              Records that failed to export
            </div>
          </div>
        );
      },
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={TriangleAlert}
      state={useCollectorMetricsState}
      title={"Failed Records"}
    />
  );
};

export default CollectorExporterTabContent;
