import { JSX } from "react";
import { Pod } from "../models";
import { TabsContent } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Activity, Clock, Database, FileText } from "lucide-react";
import StatRow from "@/components/shared/StatRow";
import { METRICS } from "@/constants/metrics.constants";
import { bytesToUnits, formatTime } from "@/utils/formatter";

const CollectorProcessTab = ({
  collector,
}: {
  collector: Pod;
}): JSX.Element => {
  const uptime: number = collector.getMetric(
    METRICS.OTELCOL_PROCESS_UPTIME_SECONDS
  );

  const cpuSeconds: number = collector.getMetric(
    METRICS.OTELCOL_PROCESS_CPU_SECONDS
  );

  const memoryRss: number = collector.getMetric(
    METRICS.OTELCOL_PROCESS_MEMORY_RSS
  );
  const heapAlloc: number = collector.getMetric(
    METRICS.OTELCOL_PROCESS_RUNTIME_HEAP_ALLOC
  );
  const sysMemory: number = collector.getMetric(
    METRICS.OTELCOL_PROCESS_RUNTIME_TOTAL_SYS_MEMORY
  );
  const totalAlloc: number = collector.getMetric(
    METRICS.OTELCOL_PROCESS_RUNTIME_TOTAL_ALLOC
  );

  const openFilesRatio: number = collector.getMetric(
    METRICS.OTELCOL_FILECONSUMER_OPEN_FILES
  );
  const readingFilesRatio: number = collector.getMetric(
    METRICS.OTELCOL_FILECONSUMER_READING_FILES
  );

  return (
    <TabsContent
      value="process"
      className="grid gap-6 md:grid-cols-2 lg:grid-cols-3"
    >
      <CPUStatsCard cpuSeconds={cpuSeconds} />
      <UptimeCard uptime={uptime} />
      <FileStatsCard
        openFilesRatio={openFilesRatio}
        readingFilesRatio={readingFilesRatio}
      />
      <MemoryStatsCard
        memoryRss={memoryRss}
        heapAlloc={heapAlloc}
        sysMemory={sysMemory}
        totalAlloc={totalAlloc}
      />
    </TabsContent>
  );
};

export default CollectorProcessTab;

const CPUStatsCard = ({ cpuSeconds }: { cpuSeconds: number }): JSX.Element => {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Activity className="h-5 w-5" />
          CPU Usage
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{cpuSeconds.toFixed(2)}s</div>
        <p className="text-xs text-muted-foreground">Total CPU time consumed</p>
      </CardContent>
    </Card>
  );
};

const UptimeCard = ({ uptime }: { uptime: number }): JSX.Element => {
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

const MemoryStatsCard = ({
  memoryRss,
  heapAlloc,
  sysMemory,
  totalAlloc,
}: {
  memoryRss: number;
  heapAlloc: number;
  sysMemory: number;
  totalAlloc: number;
}): JSX.Element => {
  const memoryRssInUnits = bytesToUnits(memoryRss);
  const heapAllocInUnits = bytesToUnits(heapAlloc);
  const sysMemoryInUnits = bytesToUnits(sysMemory);
  const totalAllocInUnits = bytesToUnits(totalAlloc);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Database className="h-5 w-5" />
          Memory
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <StatRow text="RSS" value={memoryRssInUnits} />
        <StatRow text="Heap Alloc" value={heapAllocInUnits} />
        <StatRow text="Sys Memory" value={sysMemoryInUnits} />
        <StatRow text="Total Alloc" value={totalAllocInUnits} />
      </CardContent>
    </Card>
  );
};

const FileStatsCard = ({
  openFilesRatio,
  readingFilesRatio,
}: {
  openFilesRatio: number;
  readingFilesRatio: number;
}): JSX.Element => {
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
