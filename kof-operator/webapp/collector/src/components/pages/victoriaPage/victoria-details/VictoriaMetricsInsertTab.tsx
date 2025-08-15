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
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
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
      metricName: VICTORIA_METRICS.VM_RPC_VMSTORAGE_IS_REACHABLE.name,
      metricFormat: (value) => (value == 0 ? "False" : "True"),
      customRow: ({ rawValue, title, formattedValue }) => {
        const color = rawValue == 0 ? "text-red-600" : "text-green-600";
        return (
          <StatRow
            key={title}
            value={formattedValue}
            valueStyles={color}
            text={title}
            hint={VICTORIA_METRICS.VM_RPC_VMSTORAGE_IS_REACHABLE.hint}
          ></StatRow>
        );
      },
    },
    {
      title: "VMStorage read-only",
      metricName: VICTORIA_METRICS.VM_RPC_VMSTORAGE_IS_READ_ONLY.name,
      metricFormat: (value) => (value == 0 ? "False" : "True"),
      customRow: ({ rawValue, title, formattedValue }) => {
        const color = rawValue == 1 ? "text-red-600" : "text-green-600";
        return (
          <StatRow
            key={title}
            value={formattedValue}
            valueStyles={color}
            text={title}
            hint={VICTORIA_METRICS.VM_RPC_VMSTORAGE_IS_READ_ONLY.hint}
          ></StatRow>
        );
      },
    },
    {
      title: "Dial errors",
      metricName: VICTORIA_METRICS.VM_RPC_DIAL_ERRORS_TOTAL.name,
      hint: VICTORIA_METRICS.VM_RPC_DIAL_ERRORS_TOTAL.hint,
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
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_DROPPED_ON_OVERLOAD_TOTAL.name,
      hint: VICTORIA_METRICS.VM_RPC_ROWS_DROPPED_ON_OVERLOAD_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Incompletely replicated",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_INCOMPLETELY_REPLICATED_TOTAL.name,
      hint: VICTORIA_METRICS.VM_RPC_ROWS_INCOMPLETELY_REPLICATED_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Invalid rows",
      metricName: VICTORIA_METRICS.VM_ROWS_INVALID_TOTAL.name,
      hint: VICTORIA_METRICS.VM_ROWS_INVALID_TOTAL.hint,
      enableTrendSystem: true,
      isPositiveTrend: false,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Reroutes",
      metricName: VICTORIA_METRICS.VM_RPC_REROUTES_TOTAL.name,
      hint: VICTORIA_METRICS.VM_RPC_REROUTES_TOTAL.hint,
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
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_SENT_TOTAL.name,
      hint: VICTORIA_METRICS.VM_RPC_ROWS_SENT_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows pushed",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_PUSHED_TOTAL.name,
      hint: VICTORIA_METRICS.VM_RPC_ROWS_PUSHED_TOTAL.hint,
      enableTrendSystem: true,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Rows pending",
      metricName: VICTORIA_METRICS.VM_RPC_ROWS_PENDING.name,
      hint: VICTORIA_METRICS.VM_RPC_ROWS_PENDING.hint,
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
