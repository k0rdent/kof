import { TabsContent } from "@/components/generated/ui/tabs";
import { JSX } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import StatRowWithTrend from "@/components/shared/StatRowWithTrend";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";
import { getMetricTrendData } from "@/utils/metrics";
import StatRow from "@/components/shared/StatRow";

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
  const { metricsHistory, selectedCollector: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const { metricValue: batchSendSize, metricTrend: trendSendSize } =
    getMetricTrendData(
      METRICS.OTELCOL_PROCESSOR_BATCH_SEND_SIZE,
      metricsHistory,
      col,
      timePeriod
    );

  const {
    metricValue: batchSizeTriggerSend,
    metricTrend: trendSizeTriggerSend,
  } = getMetricTrendData(
    METRICS.OTELCOL_PROCESSOR_BATCH_SIZE_TRIGGER_SEND,
    metricsHistory,
    col,
    timePeriod
  );

  const {
    metricValue: batchTimeoutTriggerSend,
    metricTrend: trendTimeoutTriggerSend,
  } = getMetricTrendData(
    METRICS.OTELCOL_PROCESSOR_BATCH_TIMEOUT_TRIGGER_SEND,
    metricsHistory,
    col,
    timePeriod
  );

  const batchMetadataCardinality: number = col.getMetric(
    METRICS.OTELCOL_PROCESSOR_BATCH_METADATA_CARDINALITY
  );

  const formattedSendSize = formatNumber(batchSendSize);
  const formattedSizeTrigger = formatNumber(batchSizeTriggerSend);
  const formattedTimeoutTrigger = formatNumber(batchTimeoutTriggerSend);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Batch processor performance metrics</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRowWithTrend
          text="Batch Send Size"
          value={formattedSendSize}
          trend={trendSendSize}
        />
        <StatRowWithTrend
          text="Size Trigger Sends"
          value={formattedSizeTrigger}
          trend={trendSizeTriggerSend}
        />
        <StatRowWithTrend
          text="Timeout Trigger Sends"
          value={formattedTimeoutTrigger}
          trend={trendTimeoutTriggerSend}
        />
        <StatRow text="Metadata Cardinality" value={batchMetadataCardinality} />
      </CardContent>
    </Card>
  );
};

const ItemFlowCard = (): JSX.Element => {
  const { metricsHistory, selectedCollector: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const { metricValue: incoming, metricTrend: trendIn } = getMetricTrendData(
    METRICS.OTELCOL_PROCESSOR_INCOMING_ITEMS,
    metricsHistory,
    col,
    timePeriod
  );

  const { metricValue: outgoing, metricTrend: trendOut } = getMetricTrendData(
    METRICS.OTELCOL_PROCESSOR_OUTGOING_ITEMS,
    metricsHistory,
    col,
    timePeriod
  );

  const formattedIncomingItems = formatNumber(incoming);
  const formattedOutgoingItems = formatNumber(outgoing);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Items processed through the pipeline</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRowWithTrend
          text="Incoming Items"
          value={formattedIncomingItems}
          trend={trendIn}
          isPositiveTrend={true}
        />
        <StatRowWithTrend
          text="Outgoing Items"
          value={formattedOutgoingItems}
          trend={trendOut}
          isPositiveTrend={true}
        />
      </CardContent>
    </Card>
  );
};
