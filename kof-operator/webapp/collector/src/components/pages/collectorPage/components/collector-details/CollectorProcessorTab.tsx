import { TabsContent } from "@/components/generated/ui/tabs";
import { JSX } from "react";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { BarChart2, Shuffle } from "lucide-react";

const CollectorProcessorTab = (): JSX.Element => {
  return (
    <TabsContent value="processor" className="grid gap-6 md:grid-cols-2">
      <BatchStatsCard />
      <ItemFlowCard />
    </TabsContent>
  );
};

export default CollectorProcessorTab;

const BatchStatsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Batch Send Size",
      metricName: METRICS.OTELCOL_PROCESSOR_BATCH_SEND_SIZE.name,
      hint: METRICS.OTELCOL_PROCESSOR_BATCH_SEND_SIZE.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Size Trigger Sends",
      metricName: METRICS.OTELCOL_PROCESSOR_BATCH_SIZE_TRIGGER_SEND.name,
      hint: METRICS.OTELCOL_PROCESSOR_BATCH_SIZE_TRIGGER_SEND.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Timeout Trigger Sends",
      metricName: METRICS.OTELCOL_PROCESSOR_BATCH_TIMEOUT_TRIGGER_SEND.name,
      hint: METRICS.OTELCOL_PROCESSOR_BATCH_TIMEOUT_TRIGGER_SEND.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Metadata Cardinality",
      metricName: METRICS.OTELCOL_PROCESSOR_BATCH_METADATA_CARDINALITY.name,
      hint: METRICS.OTELCOL_PROCESSOR_BATCH_METADATA_CARDINALITY.hint,
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={BarChart2}
      state={useCollectorMetricsState}
      title="Batch processor performance metrics"
    />
  );
};

const ItemFlowCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Incoming Items",
      metricName: METRICS.OTELCOL_PROCESSOR_INCOMING_ITEMS.name,
      hint: METRICS.OTELCOL_PROCESSOR_INCOMING_ITEMS.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Outgoing Items",
      metricName: METRICS.OTELCOL_PROCESSOR_OUTGOING_ITEMS.name,
      hint: METRICS.OTELCOL_PROCESSOR_OUTGOING_ITEMS.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Shuffle}
      state={useCollectorMetricsState}
      title="Items processed through the pipeline"
    />
  );
};
