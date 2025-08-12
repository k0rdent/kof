import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { formatNumber } from "@/utils/formatter";
import { CheckCircle, Database } from "lucide-react";
import { JSX } from "react";

const VictoriaMetricsStorageTab = (): JSX.Element => {
  return (
    <TabsContent value="vm_storage" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-2">
        <OverviewCard />
        <IngestQualityCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaMetricsStorageTab;

const OverviewCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows Received",
      metricName: VICTORIA_METRICS.VM_ROWS_RECEIVED_BY_STORAGE_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Added to Storage",
      metricName: VICTORIA_METRICS.VM_ROWS_ADDED_TO_STORAGE_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Data Size",
      metricName: VICTORIA_METRICS.VM_DATA_SIZE_BYTES,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Active Merges",
      metricName: VICTORIA_METRICS.VM_ACTIVE_MERGE,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Database}
      state={useVictoriaMetricsState}
      title={"Overview"}
    />
  );
};

const IngestQualityCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows Invalid",
      metricName: VICTORIA_METRICS.VM_ROWS_INVALID_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Ignored",
      metricName: VICTORIA_METRICS.VM_ROWS_IGNORED_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Deleted",
      metricName: VICTORIA_METRICS.VM_ROWS_DELETED_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Merged",
      metricName: VICTORIA_METRICS.VM_ROWS_MERGED_TOTAL,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={CheckCircle}
      state={useVictoriaMetricsState}
      title={"Ingest Quality"}
    />
  );
};
