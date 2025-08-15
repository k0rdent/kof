import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { Activity, Zap } from "lucide-react";
import { JSX } from "react";

const VictoriaGoRuntimeTab = (): JSX.Element => {
  return (
    <TabsContent value="go_runtime" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-2">
        <GoRuntimeCard />
        <GoMemoryStatsCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaGoRuntimeTab;

const GoRuntimeCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Goroutines",
      metricName: VICTORIA_METRICS.GO_GOROUTINES.name,
      hint: VICTORIA_METRICS.GO_GOROUTINES.hint,
    },
    {
      title: "GC CPU Time",
      metricName: VICTORIA_METRICS.GO_GC_CPU_SECONDS_TOTAL.name,
      hint: VICTORIA_METRICS.GO_GC_CPU_SECONDS_TOTAL.hint,
      metricFormat: (value: number) => `${value.toFixed(2)}s`,
    },
    {
      title: "CGO Calls",
      metricName: VICTORIA_METRICS.GO_CGO_CALLS_COUNT.name,
      hint: VICTORIA_METRICS.GO_CGO_CALLS_COUNT.hint,
    },
    {
      title: "GOMAXPROCS",
      metricName: VICTORIA_METRICS.GO_GOMAXPROCES.name,
      hint: VICTORIA_METRICS.GO_GOMAXPROCES.hint,
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Zap}
      state={useVictoriaMetricsState}
      title={"Go Runtime"}
    ></MetricsCard>
  );
};

const GoMemoryStatsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Heap Allocated",
      metricName: VICTORIA_METRICS.GO_MEMSTATS_HEAP_ALLOC_BYTES.name,
      hint: VICTORIA_METRICS.GO_MEMSTATS_HEAP_ALLOC_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Heap In Use",
      metricName: VICTORIA_METRICS.GO_MEMSTATS_HEAP_INUSE_BYTES.name,
      hint: VICTORIA_METRICS.GO_MEMSTATS_HEAP_INUSE_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Heap Idle",
      metricName: VICTORIA_METRICS.GO_MEMSTATS_HEAP_IDLE_BYTES.name,
      hint: VICTORIA_METRICS.GO_MEMSTATS_HEAP_IDLE_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Heap Objects",
      metricName: VICTORIA_METRICS.GO_MEMSTATS_HEAP_OBJECTS.name,
      hint: VICTORIA_METRICS.GO_MEMSTATS_HEAP_OBJECTS.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Activity}
      state={useVictoriaMetricsState}
      title={"Go Memory Stats"}
    ></MetricsCard>
  );
};
