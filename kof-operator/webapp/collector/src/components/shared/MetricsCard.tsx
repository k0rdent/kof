import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";
import { StoreApi, UseBoundStore } from "zustand";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../generated/ui/card";
import { LucideProps } from "lucide-react";
import { DefaultProviderState } from "@/providers/DefaultProviderState";
import { useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";
import { getMetricTrendData } from "@/utils/metrics";
import StatRowWithTrend from "./StatRowWithTrend";
import StatRow from "./StatRow";
import { Pod } from "../pages/collectorPage/models";

interface CustomRowProps {
  rawValue: number;
  formattedValue: string;
  title: string;
}

export interface MetricCardRow {
  title: string;
  metricName?: string;
  metricFetchFn?: (pod: Pod) => number;
  metricFormat?: (value: number) => string;
  customRow?: (rowProps: CustomRowProps) => JSX.Element;
  enableTrendSystem?: boolean;
  isPositiveTrend?: boolean;
  hint?: string;
}

export interface MetricsCardProps {
  rows: MetricCardRow[];
  icon: ForwardRefExoticComponent<
    Omit<LucideProps, "ref"> & RefAttributes<SVGSVGElement>
  >;
  state: UseBoundStore<StoreApi<DefaultProviderState>>;
  title: string;
  description?: string;
}

export const MetricsCard = ({
  rows,
  icon: Icon,
  title,
  state,
  description,
}: MetricsCardProps): JSX.Element => {
  const { metricsHistory, selectedPod: pod } = state();
  const { timePeriod } = useTimePeriod();

  if (!pod) {
    return <></>;
  }

  const renderMetricRow = (row: MetricCardRow): JSX.Element => {
    if (row.metricFetchFn) {
      const value = row.metricFetchFn(pod);
      const formattedValue = row.metricFormat
        ? row.metricFormat(value)
        : value.toString();

      return row.customRow ? (
        row.customRow({ rawValue: value, formattedValue, title: row.title })
      ) : (
        <StatRow
          key={row.title}
          hint={row.hint}
          text={row.title}
          value={formattedValue}
        />
      );
    }

    if (!row.metricName) {
      return row.customRow ? (
        row.customRow({ rawValue: 0, formattedValue: "0", title: row.title })
      ) : (
        <StatRow key={row.title} text={row.title} value="0" hint={row.hint} />
      );
    }

    if (!row.enableTrendSystem) {
      const value = pod.getMetric(row.metricName);
      const formattedValue = row.metricFormat
        ? row.metricFormat(value)
        : value.toString();

      return row.customRow ? (
        row.customRow({ rawValue: value, formattedValue, title: row.title })
      ) : (
        <StatRow
          key={row.title}
          hint={row.hint}
          text={row.title}
          value={formattedValue}
        />
      );
    }

    const { metricValue, metricTrend } = getMetricTrendData(
      row.metricName,
      metricsHistory,
      pod,
      timePeriod
    );

    const formattedValue = row.metricFormat
      ? row.metricFormat(metricValue)
      : metricValue.toString();

    return (
      <StatRowWithTrend
        hint={row.hint}
        key={row.title}
        text={row.title}
        value={formattedValue}
        trend={metricTrend}
        isPositiveTrend={row.isPositiveTrend}
      />
    );
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center space-x-2 pb-2">
          <Icon className="h-5 w-5" />
          <CardTitle>{title}</CardTitle>
        </div>
        {description ? <CardDescription>{description}</CardDescription> : <></>}
      </CardHeader>
      <CardContent className="space-y-4">
        {rows.map((row) => {
          return renderMetricRow(row);
        })}
      </CardContent>
    </Card>
  );
};
