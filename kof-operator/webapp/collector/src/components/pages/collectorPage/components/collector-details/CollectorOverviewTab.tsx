import { JSX } from "react";
import { Pod, Metric } from "../../models";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import {
  Cpu,
  Gauge,
  MemoryStick,
  Network,
  TrendingUp,
  Zap,
} from "lucide-react";
import { TabsContent } from "@/components/generated/ui/tabs";
import { Progress } from "@/components/generated/ui/progress";
import { METRICS } from "@/constants/metrics.constants";
import { bytesToUnits, capitalizeFirstLetter, formatNumber } from "@/utils/formatter";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";
import { getMetricTrendData } from "@/utils/metrics";
import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";

const CollectorOverviewTabContent = ({
  collector,
}: {
  collector: Pod;
}): JSX.Element => {
  const memoryUsage: number =
    collector.getMetric(METRICS.CONTAINER_RESOURCE_MEMORY_USAGE.name)
      ?.totalValue ?? 0;
  const memoryLimit: number =
    collector.getMetric(METRICS.CONTAINER_RESOURCE_MEMORY_LIMIT.name)
      ?.totalValue ?? 0;

  const queueSizeMetric =
    collector.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_SIZE.name);

  const queueCapacityMetric =
    collector.getMetric(METRICS.OTELCOL_EXPORTER_QUEUE_CAPACITY.name);

  const cpuUsage =
    collector.getMetric(METRICS.CONTAINER_RESOURCE_CPU_USAGE.name)
      ?.totalValue ?? 0;
  const cpuLimit =
    collector.getMetric(METRICS.CONTAINER_RESOURCE_CPU_LIMIT.name)
      ?.totalValue ?? 0;

  return (
    <TabsContent value="overview" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <CPUUsageCard currentUsage={cpuUsage} cpuLimit={cpuLimit} />
        <MemoryUsageCard memoryUsage={memoryUsage} memoryLimit={memoryLimit} />
        <QueueCard queueSizeMetric={queueSizeMetric} queueCapacityMetric={queueCapacityMetric} />
        <MetricsStatCard />
      </div>
      <div className="grid gap-6 md:grid-cols-2">
        <ExportPerformanceCard />
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
  const usagePercentage = cpuLimit > 0 ? (currentUsage / cpuLimit) * 100 : 0;
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
  const usagePercentage =
    memoryLimit > 0 ? (memoryUsage / memoryLimit) * 100 : 0;
  const memoryUsageUnits = bytesToUnits(memoryUsage);
  const memoryLimitUnits = bytesToUnits(memoryLimit);

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
          Limit: {memoryLimitUnits} | Current: {memoryUsageUnits}
        </p>
      </CardContent>
    </Card>
  );
};

const QueueCard = ({
  queueSizeMetric,
  queueCapacityMetric,
}: {
  queueSizeMetric: Metric | undefined;
  queueCapacityMetric: Metric | undefined;
}): JSX.Element => {


  // Get all queue size metric values and match with capacities
  const queueItems = queueSizeMetric?.metricValues.map((sizeValue) => {
    const capacityValue = queueCapacityMetric?.metricValues.find(v => v.labels.data_type === sizeValue.labels.data_type);
    const utilization = capacityValue && capacityValue.numValue > 0 ? (sizeValue.numValue / capacityValue.numValue) * 100 : 0;
    return {
      size: sizeValue.numValue,
      capacity: capacityValue?.numValue ?? 0,
      utilization,
      labels: sizeValue.labels,
      id: sizeValue.id,
    };
  });

  return (
    <Card>
      <CardHeader className="flex items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">Queue Utilization</CardTitle>
        <Gauge className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="mt-2 space-y-3">
          {!queueItems && <span className="text-sm text-muted-foreground">No metrics available</span>}
          {queueItems?.map((item) => (
            <div key={item.id}>
              <div className="text-sm font-bold">{item.utilization.toFixed(1)}% {capitalizeFirstLetter(item.labels.data_type)}</div>
              <div className="space-y-1">
              <Progress value={item.utilization} className="mt-2" />
              <p className="text-xs text-muted-foreground">
                {item.size} / {item.capacity}
              </p>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
};

const MetricsStatCard = (): JSX.Element => {
  const { metricsHistory, selectedPod: col } = useCollectorMetricsState();
  const { timePeriod } = useTimePeriod();

  if (!col) {
    return <></>;
  }

  const { metricValue, metricTrend } = getMetricTrendData(
    METRICS.OTELCOL_EXPORTER_SENT_METRIC_POINTS.name,
    metricsHistory,
    col,
    timePeriod
  );

  const trendMessageColor = metricTrend?.isTrending
    ? "text-green-600"
    : "text-red-600";

  const formattedSentMetrics = formatNumber(metricValue);

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">Metric Sent</CardTitle>
        <Zap className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="flex flex-col text-2xl font-bold">
          <div
            className={`flex gap-2 items-center font-medium ${trendMessageColor}`}
          >
            {metricTrend.isTrending && <TrendingUp className="w-5 h-5" />}
            {metricTrend.message}
          </div>
          <span className="text-xl">{formattedSentMetrics}</span>
        </div>
      </CardContent>
    </Card>
  );
};

const ExportPerformanceCard = (): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "Sent Batches",
      metricName: METRICS.OTELCOL_EXPORTER_PROM_WRITE_SENT_BATCHES.name,
      hint: METRICS.OTELCOL_EXPORTER_PROM_WRITE_SENT_BATCHES.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Time Series Ratio",
      metricName: METRICS.OTELCOL_EXPORTER_PROM_WRITE_TRANS_RATIO.name,
      hint: METRICS.OTELCOL_EXPORTER_PROM_WRITE_TRANS_RATIO.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Active Consumers",
      metricName: METRICS.OTELCOL_EXPORTER_PROM_WRITE_CONSUMERS.name,
      hint: METRICS.OTELCOL_EXPORTER_PROM_WRITE_CONSUMERS.hint,
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Network}
      state={useCollectorMetricsState}
      title="Export Performance"
      description="Metrics from Prometheus Remote Write exporter"
    />
  );
};
