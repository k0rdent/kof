import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricCardRow, MetricsCard } from "@/components/shared/MetricsCard";
import StatRow from "@/components/shared/StatRow";
import { VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { formatNumber } from "@/utils/formatter";
import { Activity, AlertTriangle, ShieldCheck } from "lucide-react";
import { JSX } from "react";

const VictoriaMetricsInsertTab = (): JSX.Element => {
  return (
    <TabsContent value="vm_insert" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-2">
        <VictoriaMetricsOverviewCard />
        <InsertRPCHealth />
        <OverloadAndReplicationCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaMetricsInsertTab;

const InsertRPCHealth = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "VMStorage reachable",
      metricName: VICTORIA_METRICS.VM_RPC_VMSTORAGE_IS_REACHABLE,
      metricFormat: (value) => (value == 0 ? "False" : "True"),
      customRow: ({ rawValue, title, formattedValue }) => {
        const color = rawValue == 0 ? "text-red-600" : "text-green-600";
        return (
          <StatRow
            key={title}
            value={formattedValue}
            valueStyles={color}
            text={title}
          ></StatRow>
        );
      },
    },
    {
      title: "VMStorage read-only",
      metricName: VICTORIA_METRICS.VM_RPC_VMSTORAGE_IS_READ_ONLY,
      metricFormat: (value) => (value == 0 ? "False" : "True"),
      customRow: ({ rawValue, title, formattedValue }) => {
        const color = rawValue == 1 ? "text-red-600" : "text-green-600";
        return (
          <StatRow
            key={title}
            value={formattedValue}
            valueStyles={color}
            text={title}
          ></StatRow>
        );
      },
    },
    {
      title: "Dial errors",
      metricName: VICTORIA_METRICS.VM_RPC_DIAL_ERRORS_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={ShieldCheck}
      state={useVictoriaMetricsState}
      title={"RPC Health"}
    />
  );
};

const OverloadAndReplicationCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Dropped on overload",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_DROPPED_ON_OVERLOAD_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Incompletely replicated",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_INCOMPLETELY_REPLICATED_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Invalid rows",
      metricName: VICTORIA_METRICS.VM_ROWS_INVALID_TOTAL,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Reroutes",
      metricName: VICTORIA_METRICS.VM_RPC_REROUTES_TOTAL,
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
      title={"Overload & Replication Issues"}
    />
  );
};

const VictoriaMetricsOverviewCard = (): JSX.Element => {
  const row: MetricCardRow[] = [
    {
      title: "Rows sent",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_SENT_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows pushed",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_PUSHED_TOTAL,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows pending",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_PENDING,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={row}
      icon={Activity}
      state={useVictoriaMetricsState}
      title={"Throughput"}
    />
  );
};
