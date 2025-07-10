import { JSX } from "react";
import { Pod } from "../models";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Cpu, Gauge, MemoryStick, Zap } from "lucide-react";
import { TabsContent } from "@/components/ui/tabs";
import { Progress } from "@/components/ui/progress";
import StatRow from "@/components/shared/StatRow";
import { METRICS } from "@/constants/metrics.constants";
import { formatBytes, formatNumber } from "@/utils/formatter";

const CollectorOverviewTabContent = ({
  collector,
}: {
  collector: Pod;
}): JSX.Element => {
  const memoryUsage: number = collector.getMetric(
    METRICS.OTELCOL_CONTAINER_RESOURCE_MEMORY_USAGE
  );
  const memoryLimit: number = collector.getMetric(
    METRICS.OTELCOL_CONTAINER_RESOURCE_MEMORY_LIMIT
  );

  const queueSize = collector.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_SIZE);
  const queueCapacity = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY
  );

  const metricsTotal = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS
  );
  const failedMetricsTotal = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_SEND_FAILED_METRIC_POINTS
  );

  const consumers = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_PROM_WRITE_CONSUMERS
  );
  const batchesTotal = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_PROM_WRITE_SENT_BATCHES
  );
  const timeSeriesRatio = collector.getMetric(
    METRICS.OTELCOL_EXPORTER_PROM_WRITE_TRANS_RATIO
  );

  const cpuUsage = collector.getMetric(
    METRICS.OTELCOL_CONTAINER_RESOURCE_CPU_USAGE
  );
  const cpuLimit = collector.getMetric(
    METRICS.OTELCOL_CONTAINER_RESOURCE_CPU_LIMIT
  );

  return (
    <TabsContent value="overview" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <CPUUsageCard currentUsage={cpuUsage} cpuLimit={cpuLimit} />
        <MemoryUsageCard memoryUsage={memoryUsage} memoryLimit={memoryLimit} />
        <QueueCard queueSize={queueSize} queueCapacity={queueCapacity} />
        <MetricsStatCard
          metricsTotal={metricsTotal}
          failedMetricsTotal={failedMetricsTotal}
        />
      </div>
      <div className="grid gap-6 md:grid-cols-2">
        <ExportPerformanceCard
          consumers={consumers}
          batchesTotal={batchesTotal}
          timeSeriesRatio={timeSeriesRatio}
        />
      </div>
    </TabsContent>
  );
};
export default CollectorOverviewTabContent;

const CPUUsageCard = ({
  currentUsage,
  cpuLimit,
}: {
  currentUsage: number;
  cpuLimit: number;
}): JSX.Element => {
  const usagePercentage = (currentUsage / cpuLimit) * 100;
  const cpuLimitInCores = cpuLimit / 1000;
  const currentCpuInCores = currentUsage / 1000;

  return (
    <Card>
      <CardHeader className="flex items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">CPU Usage</CardTitle>
        <Cpu className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{usagePercentage.toFixed(1)}%</div>
        <Progress value={usagePercentage} className="mt-2" />
        <p className="text-xs text-muted-foreground mt-1">
          Limit: {cpuLimitInCores.toFixed(2)} CPU | Current:{" "}
          {currentCpuInCores.toFixed(2)} CPU
        </p>
      </CardContent>
    </Card>
  );
};

const MemoryUsageCard = ({
  memoryUsage,
  memoryLimit,
}: {
  memoryUsage: number;
  memoryLimit: number;
}): JSX.Element => {
  const usagePercentage = (memoryUsage / memoryLimit) * 100;
  const formattedMemoryUsage = formatBytes(memoryUsage);
  const formattedMemoryLimit = formatBytes(memoryLimit);

  return (
    <Card>
      <CardHeader className="flex items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">Memory Usage</CardTitle>
        <MemoryStick className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{usagePercentage.toFixed(1)}%</div>
        <Progress value={usagePercentage} className="mt-2" />
        <p className="text-xs text-muted-foreground mt-1">
          Limit: {formattedMemoryLimit} | Current: {formattedMemoryUsage}
        </p>
      </CardContent>
    </Card>
  );
};

const QueueCard = ({
  queueSize,
  queueCapacity,
}: {
  queueSize: number;
  queueCapacity: number;
}): JSX.Element => {
  const queueUtilization = (queueSize / queueCapacity) * 100;

  return (
    <Card>
      <CardHeader className="flex items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">Queue Utilization</CardTitle>
        <Gauge className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{queueUtilization.toFixed(1)}%</div>
        <Progress value={queueUtilization} className="mt-2" />
        <p className="text-xs text-muted-foreground mt-1">
          {queueSize} / {queueCapacity}
        </p>
      </CardContent>
    </Card>
  );
};

const MetricsStatCard = ({
  metricsTotal,
  failedMetricsTotal,
}: {
  metricsTotal: number;
  failedMetricsTotal: number;
}): JSX.Element => {
  const formattedSentMetrics = formatNumber(metricsTotal);
  const formattedFailedSentMetrics = formatNumber(failedMetricsTotal);

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">Metric Sent</CardTitle>
        <Zap className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{formattedSentMetrics}</div>
        <p className="text-xs text-muted-foreground">
          Failed: {formattedFailedSentMetrics}
        </p>
      </CardContent>
    </Card>
  );
};

const ExportPerformanceCard = ({
  consumers,
  batchesTotal,
  timeSeriesRatio,
}: {
  consumers: number;
  batchesTotal: number;
  timeSeriesRatio: number;
}): JSX.Element => {
  const formattedBatchesTotal = formatNumber(batchesTotal);
  const formattedTimeSeriesRatio = formatNumber(timeSeriesRatio);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Export Performance</CardTitle>
        <CardDescription>
          Metrics from Prometheus Remote Write exporter
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <StatRow text="Active Consumers" value={consumers} />
        <StatRow text="Sent Batches" value={formattedBatchesTotal} />
        <StatRow text="Time Series Ratio" value={formattedTimeSeriesRatio} />
      </CardContent>
    </Card>
  );
};