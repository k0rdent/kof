import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { Archive, Gauge, Layers } from "lucide-react";
import { JSX } from "react";

const VictoriaMetricsSelectTab = (): JSX.Element => {
  return (
    <TabsContent value="vm_select" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <TempBlocksCard />
        <RollupResultCacheCard />
        <ReadPathThroughputCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaMetricsSelectTab;

const TempBlocksCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Tmp Files Created",
      metricName: VICTORIA_METRICS.VM_TMP_BLOCK_FILES_CREATED_TOTAL.name,
      hint: VICTORIA_METRICS.VM_TMP_BLOCK_FILES_CREATED_TOTAL.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Tmp Dir Free",
      metricName: VICTORIA_METRICS.VM_TMP_BLOCK_FILES_DIRECTORY_FREE_BYTES.name,
      hint: VICTORIA_METRICS.VM_TMP_BLOCK_FILES_DIRECTORY_FREE_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Max in‑mem tmp file size",
      metricName: VICTORIA_METRICS.VM_TMP_BLOCK_MAX_INMEMORY_FILE_SIZE_BYTES.name,
      hint: VICTORIA_METRICS.VM_TMP_BLOCK_MAX_INMEMORY_FILE_SIZE_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Archive}
      state={useVictoriaMetricsState}
      title={"Temp Blocks"}
    />
  );
};

const RollupResultCacheCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Full hits",
      metricName: VICTORIA_METRICS.VM_ROLLUP_RESULT_CACHE_FULL_HITS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROLLUP_RESULT_CACHE_FULL_HITS_TOTAL.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Partial hits",
      metricName: VICTORIA_METRICS.VM_ROLLUP_RESULT_CACHE_PARTIAL_HITS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROLLUP_RESULT_CACHE_PARTIAL_HITS_TOTAL.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Misses",
      metricName: VICTORIA_METRICS.VM_ROLLUP_RESULT_CACHE_MISS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROLLUP_RESULT_CACHE_MISS_TOTAL.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Layers}
      state={useVictoriaMetricsState}
      title={"Rollup Result Cache"}
    />
  );
};

const ReadPathThroughputCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Metric rows read",
      metricName: VICTORIA_METRICS.VM_METRIC_ROWS_READ_TOTAL.name,
      hint: VICTORIA_METRICS.VM_METRIC_ROWS_READ_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows scanned",
      metricName: VICTORIA_METRICS.VM_METRIC_ROWS_READ_TOTAL.name,
      hint: VICTORIA_METRICS.VM_METRIC_ROWS_READ_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Gauge}
      state={useVictoriaMetricsState}
      title={"Rollup Result Cache"}
    />
  );
};
