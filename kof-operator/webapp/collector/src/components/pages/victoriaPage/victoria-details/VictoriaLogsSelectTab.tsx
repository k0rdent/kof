import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { TabsContent } from "@radix-ui/react-tabs";
import { AlertTriangle, Cable } from "lucide-react";
import { JSX } from "react";

const VictoriaLogsSelectTab = (): JSX.Element => {
  return (
    <TabsContent value="vl_select" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-2">
        <ConnectionsCard />
        <ErrorsCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaLogsSelectTab;

const ConnectionsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Bytes Read",
      metricName: VICTORIA_METRICS.VLSELECT_BACKEND_CONN_BYTES_READ_TOTAL.name,
      hint: VICTORIA_METRICS.VLSELECT_BACKEND_CONN_BYTES_READ_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Bytes Written",
      metricName:
        VICTORIA_METRICS.VLSELECT_BACKEND_CONN_BYTES_WRITTEN_TOTAL.name,
      hint: VICTORIA_METRICS.VLSELECT_BACKEND_CONN_BYTES_WRITTEN_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Active Connections",
      metricName: VICTORIA_METRICS.VLSELECT_BACKEND_CONNS.name,
      hint: VICTORIA_METRICS.VLSELECT_BACKEND_CONNS.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Cable}
      state={useVictoriaMetricsState}
      title={"Connections"}
    />
  );
};

const ErrorsCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Read errors",
      metricName: VICTORIA_METRICS.VLSELECT_BACKEND_CONN_READ_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VLSELECT_BACKEND_CONN_READ_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Write errors",
      metricName: VICTORIA_METRICS.VLSELECT_BACKEND_CONN_WRITE_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VLSELECT_BACKEND_CONN_WRITE_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Dial errors",
      metricName: VICTORIA_METRICS.VLSELECT_BACKEND_DIAL_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VLSELECT_BACKEND_DIAL_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={AlertTriangle}
      state={useVictoriaMetricsState}
      title={"Errors"}
    />
  );
};
