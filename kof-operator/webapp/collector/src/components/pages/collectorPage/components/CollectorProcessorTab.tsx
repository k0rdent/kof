import { TabsContent } from "@/components/generated/ui/tabs";
import { JSX } from "react";
import { Pod } from "../models";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import { Separator } from "@/components/generated/ui/separator";
import StatRow from "@/components/shared/StatRow";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";

const CollectorProcessorTab = ({
  collector,
}: {
  collector: Pod;
}): JSX.Element => {
  const batchSendSize = collector.getMetric(
    METRICS.OTELCOL_PROCESSOR_BATCH_SEND_SIZE
  );
  const batchSizeTriggerSend = collector.getMetric(
    METRICS.OTELCOL_PROCESSOR_BATCH_SIZE_TRIGGER_SEND
  );
  const batchTimeoutTriggerSend = collector.getMetric(
    METRICS.OTELCOL_PROCESSOR_BATCH_TIMEOUT_TRIGGER_SEND
  );
  const batchMetadataCardinality = collector.getMetric(
    METRICS.OTELCOL_PROCESSOR_BATCH_METADATA_CARDINALITY
  );

  const incomingItems = collector.getMetric(
    METRICS.OTELCOL_PROCESSOR_INCOMING_ITEMS
  );
  const outgoingItems = collector.getMetric(
    METRICS.OTELCOL_PROCESSOR_OUTGOING_ITEMS
  );

  return (
    <TabsContent value="processor" className="grid gap-6 md:grid-cols-2">
      <BatchStatsCard
        batchSendSize={batchSendSize}
        batchSizeTriggerSend={batchSizeTriggerSend}
        batchTimeoutTriggerSend={batchTimeoutTriggerSend}
        batchMetadataCardinality={batchMetadataCardinality}
      />
      <ItemFlowCard
        incomingItems={incomingItems}
        outgoingItems={outgoingItems}
      />
    </TabsContent>
  );
};

export default CollectorProcessorTab;

const BatchStatsCard = ({
  batchSendSize,
  batchSizeTriggerSend,
  batchTimeoutTriggerSend,
  batchMetadataCardinality,
}: {
  batchSendSize: number;
  batchSizeTriggerSend: number;
  batchTimeoutTriggerSend: number;
  batchMetadataCardinality: number;
}): JSX.Element => {
  const formBatchSendSize = formatNumber(batchSendSize);
  const formBatchSizeTriggerSend = formatNumber(batchSizeTriggerSend);
  const formBatchTimeoutTriggerSend = formatNumber(batchTimeoutTriggerSend);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Batch processor performance metrics</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow text="Batch Send Size" value={formBatchSendSize} />
        <StatRow text="Size Trigger Sends" value={formBatchSizeTriggerSend} />
        <StatRow
          text="Timeout Trigger Sends"
          value={formBatchTimeoutTriggerSend}
        />
        <StatRow text="Metadata Cardinality" value={batchMetadataCardinality} />
      </CardContent>
    </Card>
  );
};

const ItemFlowCard = ({
  incomingItems,
  outgoingItems,
}: {
  incomingItems: number;
  outgoingItems: number;
}): JSX.Element => {
  const formattedIncomingItems = formatNumber(incomingItems);
  const formattedOutgoingItems = formatNumber(outgoingItems);
  const efficiencyPercent = ((incomingItems / outgoingItems) * 100).toFixed(2);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Items processed through the pipeline</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow text="Incoming Items" value={formattedIncomingItems} />
        <StatRow text="Outgoing Items" value={formattedOutgoingItems} />
        <Separator />
        <StatRow
          text="Processing Efficiency"
          value={`${efficiencyPercent}%`}
          valueStyles="text-sm"
        />
      </CardContent>
    </Card>
  );
};
