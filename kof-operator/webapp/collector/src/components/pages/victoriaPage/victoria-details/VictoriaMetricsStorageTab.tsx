import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { Archive, CheckCircle, Database, Download, Upload } from "lucide-react";
import { JSX } from "react";

const VictoriaMetricsStorageTab = (): JSX.Element => {
  return (
    <TabsContent value="vm_storage" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-2">
        <OverviewCard />
        <IngestQualityCard />
      </div>
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <VMInsertLinkCard />
        <VMSelectLinkCard />
        <CompressionCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaMetricsStorageTab;

const OverviewCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows Received",
      metricName: VICTORIA_METRICS.VM_ROWS_RECEIVED_BY_STORAGE_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROWS_RECEIVED_BY_STORAGE_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Added to Storage",
      metricName: VICTORIA_METRICS.VM_ROWS_ADDED_TO_STORAGE_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROWS_ADDED_TO_STORAGE_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Data Size",
      metricName: VICTORIA_METRICS.VM_DATA_SIZE_BYTES.name,
      hint: VICTORIA_METRICS.VM_DATA_SIZE_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
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
      metricName: VICTORIA_METRICS.VM_ROWS_INVALID_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROWS_INVALID_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Ignored",
      metricName: VICTORIA_METRICS.VM_ROWS_IGNORED_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROWS_IGNORED_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Deleted",
      metricName: VICTORIA_METRICS.VM_ROWS_DELETED_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROWS_DELETED_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows Merged",
      metricName: VICTORIA_METRICS.VM_ROWS_MERGED_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROWS_MERGED_TOTAL.hint,
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

const VMInsertLinkCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Metrics Read",
      metricName: VICTORIA_METRICS.VM_VMINSERT_METRICS_READ_TOTAL.name,
      hint: VICTORIA_METRICS.VM_VMINSERT_METRICS_READ_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Connection errors",
      metricName: VICTORIA_METRICS.VM_VMINSERT_CONN_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_VMINSERT_CONN_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Connections",
      metricName: VICTORIA_METRICS.VM_VMINSERT_CONNS.name,
      hint: VICTORIA_METRICS.VM_VMINSERT_CONNS.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Download}
      state={useVictoriaMetricsState}
      title={"VMInsert Link"}
    />
  );
};

const VMSelectLinkCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows Read for Queries",
      metricName: VICTORIA_METRICS.VM_VMSELECT_METRIC_ROWS_READ_TOTAL.name,
      hint: VICTORIA_METRICS.VM_VMSELECT_METRIC_ROWS_READ_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Connection Errors",
      metricName: VICTORIA_METRICS.VM_VMSELECT_CONN_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_VMSELECT_CONN_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Connections",
      metricName: VICTORIA_METRICS.VM_VMSELECT_CONNS.name,
      hint: VICTORIA_METRICS.VM_VMSELECT_CONNS.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Upload}
      state={useVictoriaMetricsState}
      title={"VMSelect Link"}
    />
  );
};

const CompressionCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Original Byte",
      metricName: VICTORIA_METRICS.VM_ZSTD_BLOCK_ORIGINAL_BYTES_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ZSTD_BLOCK_ORIGINAL_BYTES_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Compressed bytes",
      metricName: VICTORIA_METRICS.VM_ZSTD_BLOCK_COMPRESSED_BYTES_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ZSTD_BLOCK_COMPRESSED_BYTES_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Archive}
      state={useVictoriaMetricsState}
      title={"Compression Activity"}
    />
  );
};
