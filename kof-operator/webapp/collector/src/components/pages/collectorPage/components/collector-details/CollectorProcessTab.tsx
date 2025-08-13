import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Clock, Database, FileText } from "lucide-react";
import { METRICS } from "@/constants/metrics.constants";
import { bytesToUnits, formatTime } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { getAverageValue } from "@/utils/metrics";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";

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
  const rows: MetricCardRow[] = [
    {
      title: "Sent Batches",
      metricName: METRICS.OTELCOL_PROCESS_UPTIME_SECONDS,
      metricFormat: (value: number) => formatTime(value),
    },
    {
      title: "Time Series Ratio",
      metricName: METRICS.OTELCOL_PROCESS_UPTIME_SECONDS,
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
      title="Export Performance"
    />
  );
};

const FileStatsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Open Files",
      metricName: METRICS.OTELCOL_FILECONSUMER_OPEN_FILES,
    },
    {
      title: "Reading Files",
      metricName: METRICS.OTELCOL_FILECONSUMER_READING_FILES,
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

  const rows: MetricCardRow[] = [
    {
      title: "RSS",
      metricFetchFn: (pod) =>
        getAverageValue(
          METRICS.OTELCOL_PROCESS_MEMORY_RSS,
          metricsHistory,
          pod,
          timePeriod
        ),
      metricFormat: (value: number) =>
        `${bytesToUnits(value)} (Avg in ${timePeriod.text})`,
    },
    {
      title: "Heap Alloc",
      metricFetchFn: (pod) =>
        getAverageValue(
          METRICS.OTELCOL_PROCESS_RUNTIME_HEAP_ALLOC,
          metricsHistory,
          pod,
          timePeriod
        ),
      metricFormat: (value: number) =>
        `${bytesToUnits(value)} (Avg in ${timePeriod.text})`,
    },

    {
      title: "Sys Memory",
      metricFetchFn: (pod) =>
        getAverageValue(
          METRICS.OTELCOL_PROCESS_RUNTIME_TOTAL_SYS_MEMORY,
          metricsHistory,
          pod,
          timePeriod
        ),
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
