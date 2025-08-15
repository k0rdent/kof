import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { Database, Globe, Network } from "lucide-react";
import { JSX } from "react";
import { Pod } from "../../collectorPage/models";
import { getVictoriaType } from "../utils";

const VictoriaNetworkTab = (): JSX.Element => {
  const { selectedPod: pod } = useVictoriaMetricsState();
  if (!pod) {
    return <></>;
  }

  const podType = getVictoriaType(pod.name);

  return (
    <TabsContent value="network" className="flex flex-col gap-5">
      <div className="grid gap-6 lg:grid-cols-2">
        <TCPConnectionDetailsCard />
        <HTTPPerformanceDetailsCard />
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {(podType === "vlinsert" ||
          podType === "vlselect" ||
          podType === "vlstorage") && <VictoriaLogsNetworkCard />}
      </div>
    </TabsContent>
  );
};

export default VictoriaNetworkTab;

const TCPConnectionDetailsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Data Read",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_READ_BYTES_TOTAL.name,
      hint: VICTORIA_METRICS.VM_TCPLISTENER_READ_BYTES_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Data Written",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_WRITTEN_BYTES_TOTAL.name,
      hint: VICTORIA_METRICS.VM_TCPLISTENER_WRITTEN_BYTES_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Total Accepts",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_ACCEPTS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_TCPLISTENER_ACCEPTS_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Active Connections",
      metricName: VICTORIA_METRICS.VM_TCPLISTENER_CONNS.name,
      hint: VICTORIA_METRICS.VM_TCPLISTENER_CONNS.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Network}
      state={useVictoriaMetricsState}
      title={"TCP Connection Details"}
    />
  );
};

const HTTPPerformanceDetailsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Total HTTP Requests",
      metricName: VICTORIA_METRICS.VM_HTTP_REQUESTS_ALL_TOTAL.name,
      hint: VICTORIA_METRICS.VM_HTTP_REQUESTS_ALL_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Request Errors",
      metricName: VICTORIA_METRICS.VM_HTTP_REQUEST_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_HTTP_REQUEST_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Timeout Closed Conns",
      metricName: VICTORIA_METRICS.VM_HTTP_CONN_TIMEOUT_CLOSED_CONNS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_HTTP_CONN_TIMEOUT_CLOSED_CONNS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Avg Response Time",
      metricFormat: (value: number) => `${value.toFixed(2)}ms`,
      metricFetchFn: (pod: Pod): number => {
        const requestDurationSec = pod.getMetric(
          VICTORIA_METRICS.VM_HTTP_REQUEST_DURATION_SECONDS_SUM.name
        );
        const requestDurationCount = pod.getMetric(
          VICTORIA_METRICS.VM_HTTP_REQUEST_DURATION_SECONDS_COUNT.name
        );
        return (requestDurationSec / requestDurationCount) * 1000;
      },
      hint: "Average response time for HTTP requests in milliseconds",
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Globe}
      state={useVictoriaMetricsState}
      title={"HTTP Performance Details"}
    />
  );
};

const VictoriaLogsNetworkCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "UDP Requests",
      metricName: VICTORIA_METRICS.VL_UDP_REQESTS_TOTAL.name,
      hint: VICTORIA_METRICS.VL_UDP_REQESTS_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "UDP Errors",
      metricName: VICTORIA_METRICS.VL_UDP_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VL_UDP_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Database}
      state={useVictoriaMetricsState}
      title={"VictoriaLogs Traffic"}
    />
  );
};
