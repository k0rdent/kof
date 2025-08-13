import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/generated/ui/card";
import { Progress } from "@/components/generated/ui/progress";
import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { METRICS, VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import {
  Activity,
  AlertTriangle,
  Cable,
  Cpu,
  Gauge,
  MemoryStick,
  Network,
  Settings,
} from "lucide-react";
import { JSX } from "react";
import { Pod } from "../../collectorPage/models";
import { getVictoriaType } from "../utils";

const VictoriaOverviewTab = (): JSX.Element => {
  const { selectedPod: pod } = useVictoriaMetricsState();
  if (!pod) {
    return <></>;
  }

  const podType = getVictoriaType(pod.name);

  return (
    <TabsContent value="overview" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <CPUUsageCard />
        <MemoryUsageCard />
        <NetworkActivityCard />
      </div>
      <div className="grid gap-6 md:grid-cols-2">
        <PerformanceCard />
        <ErrorsSummaryCard />
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {podType == "vlinsert" ||
          podType == "vlstorage" ||
          (podType == "vlselect" && <VictoriaLogsDropsCard />)}

        {podType == "vlinsert" && <VictoriaLogsInsertCard />}

        {podType == "vminsert" && <VictoriaMetricsInsertOverviewCard />}
        {podType == "vmselect" && <VictoriaMetricsSelectOverviewCard />}
      </div>
    </TabsContent>
  );
};

export default VictoriaOverviewTab;

const CPUUsageCard = (): JSX.Element => {
  const { selectedPod: pod } = useVictoriaMetricsState();

  if (!pod) {
    return <></>;
  }

  const cpuUsage = pod.getMetric(METRICS.CONTAINER_RESOURCE_CPU_USAGE);
  const cpuLimit = pod.getMetric(METRICS.CONTAINER_RESOURCE_CPU_LIMIT);

  const usagePercentage = cpuLimit > 0 ? (cpuUsage / cpuLimit) * 100 : 0;
  const cpuLimitInCores = cpuLimit / 1000;
  const currentCpuInCores = cpuUsage / 1000;

  return (
    <Card>
      <CardHeader className="flex items-center space-x-2 pb-2">
        <Cpu className="h-5 w-5" />
        <CardTitle>CPU Usage</CardTitle>
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

const MemoryUsageCard = (): JSX.Element => {
  const { selectedPod: pod } = useVictoriaMetricsState();

  if (!pod) {
    return <></>;
  }

  const memoryUsage = pod.getMetric(METRICS.CONTAINER_RESOURCE_MEMORY_USAGE);
  const memoryLimit = pod.getMetric(METRICS.CONTAINER_RESOURCE_MEMORY_LIMIT);

  const usagePercentage =
    memoryLimit > 0 ? (memoryUsage / memoryLimit) * 100 : 0;
  const memoryUsageUnits = bytesToUnits(memoryUsage);
  const memoryLimitUnits = bytesToUnits(memoryLimit);

  return (
    <Card>
      <CardHeader className="flex items-center space-x-2 pb-2">
        <MemoryStick className="h-5 w-5" />
        <CardTitle>Memory Usage</CardTitle>
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

const NetworkActivityCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Data Read",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_READ_BYTES_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Data Write",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_WRITE_BYTES_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Active Connections",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_CONNS,
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Network}
      state={useVictoriaMetricsState}
      title={"Network Activity"}
    />
  );
};

const ErrorsSummaryCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "VL Errors",
      metricName: VICTORIA_METRICS.VL_ERRORS_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "HTTP Errors",
      metricName: VICTORIA_METRICS.VL_HTTP_ERRORS_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "TCP Listener Errors",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_ERRORS_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={AlertTriangle}
      state={useVictoriaMetricsState}
      title={"Error Summary"}
    />
  );
};

const PerformanceCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Total HTTP Requests",
      metricName: VICTORIA_METRICS.VM_HTTP_REQUESTS_ALL_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Request Errors",
      metricName: VICTORIA_METRICS.VM_HTTP_REQUEST_ERRORS_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Avg Response Time",
      metricFormat: (value: number) => `${value.toFixed(2)}ms`,
      metricFetchFn: (pod: Pod): number => {
        const requestDurationSec = pod.getMetric(
          VICTORIA_METRICS.VM_HTTP_REQUEST_DURATION_SECONDS_SUM
        );
        const requestDurationCount = pod.getMetric(
          VICTORIA_METRICS.VM_HTTP_REQUEST_DURATION_SECONDS_COUNT
        );
        return (requestDurationSec / requestDurationCount) * 1000;
      },
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Activity}
      state={useVictoriaMetricsState}
      title={"Request Performance"}
    />
  );
};

const VictoriaLogsInsertCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Bytes written",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_BYTE_WRITTEN,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Bytes read",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_BYTE_READ,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Active connections",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS,
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Cable}
      state={useVictoriaMetricsState}
      title={"VictoriaLogs Insert: Overview"}
    />
  );
};

const VictoriaLogsDropsCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows dropped",
      metricName: VICTORIA_METRICS.VL_ROWS_DROPPED_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Too-long lines skipped",
      metricName: VICTORIA_METRICS.VL_TOO_LONG_LINES_SKIPPED_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Settings}
      state={useVictoriaMetricsState}
      title={"VictoriaLogs: Rows Drops"}
    />
  );
};

const VictoriaMetricsInsertOverviewCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows Sent",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_SENT_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Invalid Rows",
      metricName: VICTORIA_METRICS.VM_ROWS_INVALID_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Settings}
      state={useVictoriaMetricsState}
      title={"VictoriaMetrics Insert: Overview"}
    />
  );
};

const VictoriaMetricsSelectOverviewCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Select requests",
      metricName: VICTORIA_METRICS.VM_TENANT_SELECT_REQUEST_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows read",
      metricName: VICTORIA_METRICS.VM_METRIC_ROWS_READ_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows skipped",
      metricName: VICTORIA_METRICS.VM_METRICS_ROWS_SKIPPED_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Gauge}
      state={useVictoriaMetricsState}
      title={"VictoriaMetrics Select: Overview"}
    />
  );
};
