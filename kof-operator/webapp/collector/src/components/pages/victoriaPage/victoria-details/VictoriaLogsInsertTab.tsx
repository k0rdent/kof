import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { Activity, AlertTriangle, ShieldCheck } from "lucide-react";
import { JSX } from "react";

const VictoriaLogsInsertTab = (): JSX.Element => {
  return (
    <TabsContent value="vl_insert" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <ThroughputCard />
        <DialHealthCard />
        <IOHealthCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaLogsInsertTab;

const ThroughputCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Bytes written",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_BYTE_WRITTEN.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_BYTE_WRITTEN.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Bytes read",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_BYTE_READ.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_BYTE_READ.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Writes",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_WRITES_TOTAL.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_WRITES_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Reads",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_READS_TOTAL.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_READS_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Activity}
      state={useVictoriaMetricsState}
      title={"Throughput"}
    />
  );
};

const DialHealthCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Dial attempts",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_DIALS_TOTAL.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_DIALS_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Dial errors",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_DIALS_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_DIALS_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Dial success rate",
      metricFetchFn: (pod) => {
        const total = pod.getMetric(
          VICTORIA_METRICS.VLINSERT_BACKEND_DIALS_TOTAL.name
        );
        const error = pod.getMetric(
          VICTORIA_METRICS.VLINSERT_BACKEND_DIALS_ERRORS_TOTAL.name
        );
        return ((total - error) / total) * 100;
      },
      metricFormat: (value: number) => `${value.toFixed(2)}%`,
      hint: "Percentage of successful backend dial attempts",
    },
    {
      title: "Active connections",
      metricName: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS.hint,
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={ShieldCheck}
      state={useVictoriaMetricsState}
      title={"Dial Health"}
    />
  );
};

const IOHealthCard = (): JSX.Element => {
  const rows: MetricCardRow[] = [
    {
      title: "Read errors",
      metricName:
        VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_READ_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_READ_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Write errors",
      metricName:
        VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_WRITE_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_WRITE_ERRORS_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "I/O error ratio",
      metricFetchFn: (pod) => {
        const readErrors = pod.getMetric(
          VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_READ_ERRORS_TOTAL.name
        );
        const writeErrors = pod.getMetric(
          VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_WRITE_ERRORS_TOTAL.name
        );
        const writeTotal = pod.getMetric(
          VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_WRITES_TOTAL.name
        );
        const readTotal = pod.getMetric(
          VICTORIA_METRICS.VLINSERT_BACKEND_CONNS_READS_TOTAL.name
        );
        return ((readErrors + writeErrors) / (writeTotal + readTotal)) * 100;
      },
      metricFormat: (value: number) => `${value.toFixed(2)}%`,
      hint: "Percentage of backend I/O operations that failed",
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={AlertTriangle}
      state={useVictoriaMetricsState}
      title={"I/O Health"}
    />
  );
};
