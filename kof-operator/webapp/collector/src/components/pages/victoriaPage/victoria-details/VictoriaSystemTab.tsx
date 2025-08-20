import { TabsContent } from "@/components/generated/ui/tabs";
import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";
import { METRICS, VICTORIA_METRICS } from "@/constants/metrics.constants";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import { Cpu, HardDrive, MemoryStick } from "lucide-react";
import { JSX } from "react";

const VictoriaSystemTab = (): JSX.Element => {
  return (
    <TabsContent value="system" className="flex flex-col gap-5">
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <CpuMetricsCard />
        <MemoryMetricsCard />
        <ProcessIOActivityCard />
      </div>
    </TabsContent>
  );
};

export default VictoriaSystemTab;

const CpuMetricsCard = (): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "CPU Cores Available",
      metricName: VICTORIA_METRICS.VM_AVAILABLE_CPU_CORES.name,
      hint: VICTORIA_METRICS.VM_AVAILABLE_CPU_CORES.hint,
    },
    {
      title: "Total CPU Time",
      metricName: VICTORIA_METRICS.PROCESS_CPU_SECONDS_TOTAL.name,
      metricFormat: (value: number) => `${value.toFixed(2)}s`,
      hint: VICTORIA_METRICS.PROCESS_CPU_SECONDS_TOTAL.hint,
    },
    {
      title: "User CPU Time",
      metricName: VICTORIA_METRICS.PROCESS_CPU_SECONDS_USER_TOTAL.name,
      metricFormat: (value: number) => `${value.toFixed(2)}s`,
      hint: VICTORIA_METRICS.PROCESS_CPU_SECONDS_USER_TOTAL.hint,
    },
    {
      title: "System CPU Time",
      metricName: VICTORIA_METRICS.PROCESS_CPU_SECONDS_SYSTEM_TOTAL.name,
      metricFormat: (value: number) => `${value.toFixed(2)}s`,
      hint: VICTORIA_METRICS.PROCESS_CPU_SECONDS_SYSTEM_TOTAL.hint,
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={Cpu}
      state={useVictoriaMetricsState}
      title={"CPU Usage"}
    ></MetricsCard>
  );
};

const MemoryMetricsCard = (): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "Resident Memory",
      metricName: VICTORIA_METRICS.PROCESS_RESIDENT_MEMORY_BYTES.name,
      hint: VICTORIA_METRICS.PROCESS_RESIDENT_MEMORY_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Peak Memory",
      metricName: VICTORIA_METRICS.PROCESS_RESIDENT_MEMORY_PEAK_BYTES.name,
      hint: VICTORIA_METRICS.PROCESS_RESIDENT_MEMORY_PEAK_BYTES.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Memory Usage",
      metricName: METRICS.CONTAINER_RESOURCE_MEMORY_USAGE.name,
      hint: METRICS.CONTAINER_RESOURCE_MEMORY_USAGE.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Memory Limit",
      metricName: METRICS.CONTAINER_RESOURCE_MEMORY_LIMIT.name,
      hint: METRICS.CONTAINER_RESOURCE_MEMORY_LIMIT.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={MemoryStick}
      state={useVictoriaMetricsState}
      title={"Process Memory"}
    ></MetricsCard>
  );
};

const ProcessIOActivityCard = (): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "Read Bytes",
      metricName: VICTORIA_METRICS.PROCESS_IO_READ_BYTES_TOTAL.name,
      hint: VICTORIA_METRICS.PROCESS_IO_READ_BYTES_TOTAL.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Written Bytes",
      metricName: VICTORIA_METRICS.PROCESS_IO_WRITTEN_BYTES_TOTAL.name,
      hint: VICTORIA_METRICS.PROCESS_IO_WRITTEN_BYTES_TOTAL.hint,
      metricFormat: (value: number) => bytesToUnits(value),
    },
    {
      title: "Read Syscalls",
      metricName: VICTORIA_METRICS.PROCESS_IO_READ_SYSCALLS_TOTAL.name,
      hint: VICTORIA_METRICS.PROCESS_IO_READ_SYSCALLS_TOTAL.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
    {
      title: "Write Syscalls",
      metricName: VICTORIA_METRICS.PROCESS_IO_WRITE_SYSCALLS_TOTAL.name,
      hint: VICTORIA_METRICS.PROCESS_IO_WRITE_SYSCALLS_TOTAL.hint,
      metricFormat: (value: number) => formatNumber(value),
    },
  ];

  return (
    <MetricsCard
      rows={rows}
      icon={HardDrive}
      state={useVictoriaMetricsState}
      title={"Process I/O Activity"}
    />
  );
};
