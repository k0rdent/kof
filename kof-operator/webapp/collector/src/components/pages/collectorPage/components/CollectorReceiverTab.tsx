import { JSX } from "react";
import { Pod } from "../models";
import { TabsContent } from "@/components/generated/ui/tabs";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import StatRow from "@/components/shared/StatRow";
import { METRICS } from "@/constants/metrics.constants";
import { formatNumber } from "@/utils/formatter";

const CollectorReceiverTab = ({
  collector,
}: {
  collector: Pod;
}): JSX.Element => {
  const logRecordsReceived: number = collector.getMetric(
    METRICS.OTELCOL_RECEIVER_ACCEPTED_LOG_RECORDS
  );
  const metricRecordsReceived: number = collector.getMetric(
    METRICS.OTELCOL_RECEIVER_ACCEPTED_METRIC_POINTS
  );

  const refusedLogRecords: number = collector.getMetric(
    METRICS.OTELCOL_RECEIVER_REFUSED_LOG_RECORDS
  );
  const refusedMetricRecords: number = collector.getMetric(
    METRICS.OTELCOL_RECEIVER_REFUSED_METRIC_POINTS
  );

  return (
    <TabsContent value="receiver">
      <div className="grid gap-6 md:grid-cols-2">
        <AcceptedRecordsCard
          logRecordsReceived={logRecordsReceived}
          metricRecordsReceived={metricRecordsReceived}
        />
        <RefusedRecordsCard
          refusedLogRecords={refusedLogRecords}
          refusedMetricRecords={refusedMetricRecords}
        />
      </div>
    </TabsContent>
  );
};

export default CollectorReceiverTab;

const AcceptedRecordsCard = ({
  logRecordsReceived,
  metricRecordsReceived,
}: {
  logRecordsReceived: number;
  metricRecordsReceived: number;
}): JSX.Element => {
  const formattedLogRecordsReceived = formatNumber(logRecordsReceived);
  const formattedMetricRecordsReceived = formatNumber(metricRecordsReceived);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Successfully Received Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow text="Log Records" value={formattedLogRecordsReceived} />
        <StatRow text="Metric Points" value={formattedMetricRecordsReceived} />
      </CardContent>
    </Card>
  );
};

const RefusedRecordsCard = ({
  refusedLogRecords,
  refusedMetricRecords,
}: {
  refusedLogRecords: number;
  refusedMetricRecords: number;
}): JSX.Element => {
  const formattedRefusedLogRecords = formatNumber(refusedLogRecords);
  const formattedRefusedMetricRecords = formatNumber(refusedMetricRecords);
  const refusedLogsRecordsStyle =
    refusedLogRecords > 0 ? "text-red-600" : "text-green-600";
  const refusedMetricRecordsStyle =
    refusedMetricRecords > 0 ? "text-red-600" : "text-green-600";

  return (
    <Card>
      <CardHeader>
        <CardTitle>Rejected or Failed Records</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow
          text="Log Records"
          value={formattedRefusedLogRecords}
          valueStyles={refusedLogsRecordsStyle}
        />
        <StatRow
          text="Metric Points"
          value={formattedRefusedMetricRecords}
          valueStyles={refusedMetricRecordsStyle}
        />
      </CardContent>
    </Card>
  );
};
