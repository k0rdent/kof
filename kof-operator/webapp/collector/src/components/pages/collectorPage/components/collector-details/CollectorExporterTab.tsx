import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Progress } from "@/components/generated/ui/progress";
import { Separator } from "@/components/generated/ui/separator";
import { METRICS } from "@/constants/metrics.constants";
import { capitalizeFirstLetter, formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";
import { Clock, Send, TriangleAlert } from "lucide-react";
import { MetricValue } from "@/components/pages/collectorPage/models";

const CollectorExporterTabContent = (): JSX.Element => {
  const { selectedPod: pod } = useCollectorMetricsState();

  const capacityMetric = pod?.getMetric(
    METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.name
  );
  const sizeMetric = pod?.getMetric(
    METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.name
  );

  // Render a QueueCard for each matching pair
  const queueCards = capacityMetric?.metricValues.map((capacityValue) => {
    const sizeValue = sizeMetric?.metricValues.find(v => v.labels.data_type === capacityValue.labels.data_type);

    return (
      <QueueCard
        key={capacityValue.id}
        capacityValue={capacityValue}
        sizeValue={sizeValue}
        title={`${capitalizeFirstLetter(capacityValue.labels.data_type)} Queue Status`}
      />
    );
  });

  return (
    <TabsContent
      value="exporter"
      className="grid gap-6 md:grid-cols-2 lg:grid-cols-3"
    >
      {queueCards}
      <SentRecordsCard />
      <FailedRecordsCard />
    </TabsContent>
  );
};

type QueueCardProps = {
  capacityValue?: MetricValue;
  sizeValue?: MetricValue;
  title: string;
};

const QueueCard = ({
  capacityValue,
  sizeValue,
  title,
}: QueueCardProps): JSX.Element => {

  const cap = capacityValue?.numValue ?? 0;
  const size = sizeValue?.numValue ?? 0;
  const utilization = cap > 0 ? (size / cap) * 100 : 0;

  const rows: MetricRow[] = [
    {
      title: "Capacity",
      metricFetchFn: () => cap,
      hint: METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.hint,
    },
    {
      title: "Current Size",
      metricFetchFn: () => size,
      hint: METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.hint,
    },
    {
      title: "Utilization",
      metricFetchFn: () => utilization,
      metricFormat: (val) => `${val.toFixed(1)}%`,
      hint: "Percentage of the exporter queue currently in use",
    },
    {
      title: "Utilization Bar",
      metricFetchFn: () => utilization,
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
      title={title}
    />
  );
};

const SentRecordsCard = (): JSX.Element => {
  const rows: MetricRow[] = [
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
  const rows: MetricRow[] = [
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
