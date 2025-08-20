import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Clock, Database, FileText } from "lucide-react";
import { METRICS } from "@/constants/metrics.constants";
import { bytesToUnits, formatTime } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { getAverageValue } from "@/utils/metrics";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";
import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";

const CollectorProcessTab = (): JSX.Element => {
  return (
    <TabsContent
      value="process"
      className="grid gap-6 md:grid-cols-2 lg:grid-cols-3"
    >
      <UptimeCard />
      <FileStatsCard />
      <MemoryStatsCard />
    </TabsContent>
  );
};

export default CollectorProcessTab;

const UptimeCard = (): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "Process Uptime",
      metricName: METRICS.OTELCOL_PROCESS_UPTIME_SECONDS.name,
      hint: METRICS.OTELCOL_PROCESS_UPTIME_SECONDS.hint,
      metricFormat: (value: number) => formatTime(value),
    },
    {
      title: "Time Series Ratio",
      metricName: METRICS.OTELCOL_PROCESS_UPTIME_SECONDS.name,
      hint: METRICS.OTELCOL_PROCESS_UPTIME_SECONDS.hint,
      metricFormat: (value) => Math.round(value).toString(),
      customRow: ({ formattedValue, title }) => {
        return (
          <div key={title}>
            <p className="text-xs text-muted-foreground">
              {`${formattedValue} seconds total`}
            </p>
          </div>
        );
      },
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Clock}
      state={useCollectorMetricsState}
      title="Uptime"
    />
  );
};

const FileStatsCard = (): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "Open Files",
      metricName: METRICS.OTELCOL_FILECONSUMER_OPEN_FILES.name,
      hint: METRICS.OTELCOL_FILECONSUMER_OPEN_FILES.hint,
    },
    {
      title: "Reading Files",
      metricName: METRICS.OTELCOL_FILECONSUMER_READING_FILES.name,
      hint: METRICS.OTELCOL_FILECONSUMER_READING_FILES.hint,
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={FileText}
      state={useCollectorMetricsState}
      title="File Consumer"
    />
  );
};

const MemoryStatsCard = (): JSX.Element => {
  const { timePeriod } = useTimePeriod();
  const { metricsHistory } = useCollectorMetricsState();

  const rows: MetricRow[] = [
    {
      title: "RSS",
      metricFetchFn: (pod) =>
        getAverageValue(
          METRICS.OTELCOL_PROCESS_MEMORY_RSS.name,
          metricsHistory,
          pod,
          timePeriod
        ),
      hint: METRICS.OTELCOL_PROCESS_MEMORY_RSS.hint,
      metricFormat: (value: number) =>
        `${bytesToUnits(value)} (Avg in ${timePeriod.text})`,
    },
    {
      title: "Heap Alloc",
      metricFetchFn: (pod) =>
        getAverageValue(
          METRICS.OTELCOL_PROCESS_RUNTIME_HEAP_ALLOC.name,
          metricsHistory,
          pod,
          timePeriod
        ),
      hint: METRICS.OTELCOL_PROCESS_RUNTIME_HEAP_ALLOC.hint,
      metricFormat: (value: number) =>
        `${bytesToUnits(value)} (Avg in ${timePeriod.text})`,
    },

    {
      title: "Sys Memory",
      metricFetchFn: (pod) =>
        getAverageValue(
          METRICS.OTELCOL_PROCESS_RUNTIME_TOTAL_SYS_MEMORY.name,
          metricsHistory,
          pod,
          timePeriod
        ),
      hint: METRICS.OTELCOL_PROCESS_RUNTIME_TOTAL_SYS_MEMORY.hint,
      metricFormat: (value: number) =>
        `${bytesToUnits(value)} (Avg in ${timePeriod.text})`,
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Database}
      state={useCollectorMetricsState}
      title={"Memory"}
    />
  );
};
