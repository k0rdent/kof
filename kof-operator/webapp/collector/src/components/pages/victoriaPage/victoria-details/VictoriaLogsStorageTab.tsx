import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import StatRow from "@/components/shared/StatRow";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { Boxes, Database, HardDrive } from "lucide-react";
import { JSX } from "react";

const VictoriaLogsStorageTab = (): JSX.Element => {
  return (
    <TabsContent value="vl_storage" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <OverviewCard />
        <StorageCard />
        <StorageObjectsCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaLogsStorageTab;

const OverviewCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows ingested",
      metricName: VICTORIA_METRICS.VL_ROWS_INGESTED_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Bytes ingested",
      metricName: VICTORIA_METRICS.VL_BYTES_INGESTED_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Total errors",
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricName: VICTORIA_METRICS.VL_HTTP_ERRORS_TOTAL,
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

const StorageObjectsCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Storage parts",
      metricName: VICTORIA_METRICS.VL_STORAGE_PARTS,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Storage rows",
      metricName: VICTORIA_METRICS.VL_STORAGE_ROWS,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Storage blocks",
      metricName: VICTORIA_METRICS.VL_STORAGE_BLOCKS,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Boxes}
      state={useVictoriaMetricsState}
      title={"Storage Objects"}
    />
  );
};

const StorageCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Data Size",
      metricName: VICTORIA_METRICS.VL_DATA_SIZE_BYTES,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Free Disk Space",
      metricName: VICTORIA_METRICS.VL_FREE_DISK_SPACE_BYTES,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Read-only mode",
      metricName: VICTORIA_METRICS.VL_STORAGE_IS_READ_ONLY,
      metricFormat: (value) => (value == 0 ? "False" : "True"),
      customRow: ({ title, formattedValue }) => (
        <StatRow key={title} value={formattedValue} text={title}></StatRow>
      ),
    },
    {
      title: "Partitions",
      metricName: VICTORIA_METRICS.VL_PARTITIONS,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={HardDrive}
      state={useVictoriaMetricsState}
      title={"Storage Capacity"}
    />
  );
};
