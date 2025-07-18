import { JSX } from "react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/generated/ui/card";
import { Clock, Database, FileText } from "lucide-react";
import { METRICS } from "@/constants/metrics.constants";
import { bytesToUnits, formatTime } from "@/utils/formatter";
import StatRow from "@/components/shared/StatRow";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { getAverageValue } from "@/utils/metrics";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";

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
  const { selectedCollector: collector } = useCollectorMetricsState();

  if (!collector) {
    return <></>;
  }

  const uptime: number = collector.getMetric(
    METRICS.OTELCOL_PROCESS_UPTIME_SECONDS
  );

  const formateTime = formatTime(uptime);
  const roundedUptime = Math.round(uptime);

  return (
    <Card>
      <CardHeader className="flex items-center justify-between space-y-0 pb-2">
        <CardTitle className="flex items-center gap-2">
          <Clock className="h-5 w-5" />
          Uptime
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{formateTime}</div>
        <p className="text-xs text-muted-foreground">
          {`${roundedUptime} seconds total`}
        </p>
      </CardContent>
    </Card>
  );
};

const MemoryStatsCard = (): JSX.Element => {
  const { metricsHistory, selectedCollector: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const rssAvg = getAverageValue(
    METRICS.OTELCOL_PROCESS_MEMORY_RSS,
    metricsHistory,
    col,
    timePeriod
  );

  const heapAvg = getAverageValue(
    METRICS.OTELCOL_PROCESS_RUNTIME_HEAP_ALLOC,
    metricsHistory,
    col,
    timePeriod
  );

  const sysMemoryAvg = getAverageValue(
    METRICS.OTELCOL_PROCESS_RUNTIME_TOTAL_SYS_MEMORY,
    metricsHistory,
    col,
    timePeriod
  );

  const memoryRssInUnits = bytesToUnits(rssAvg);
  const heapAllocInUnits = bytesToUnits(heapAvg);
  const sysMemoryInUnits = bytesToUnits(sysMemoryAvg);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Database className="h-5 w-5" />
          Memory
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <StatRow
          text="RSS"
          value={`${memoryRssInUnits} (Avg in ${timePeriod.text})`}
        />
        <StatRow
          text="Heap Alloc"
          value={`${heapAllocInUnits} (Avg in ${timePeriod.text})`}
        />
        <StatRow
          text="Sys Memory"
          value={`${sysMemoryInUnits} (Avg in ${timePeriod.text})`}
        />
      </CardContent>
    </Card>
  );
};

const FileStatsCard = (): JSX.Element => {
  const { selectedCollector: col } = useCollectorMetricsState();

  if (!col) {
    return <></>;
  }

  const openFilesRatio: number = col.getMetric(
    METRICS.OTELCOL_FILECONSUMER_OPEN_FILES
  );
  const readingFilesRatio: number = col.getMetric(
    METRICS.OTELCOL_FILECONSUMER_READING_FILES
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FileText className="h-5 w-5" />
          File Consumer
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow text="Open Files" value={openFilesRatio} />
        <StatRow text="Reading Files" value={readingFilesRatio} />
      </CardContent>
    </Card>
  );
};
