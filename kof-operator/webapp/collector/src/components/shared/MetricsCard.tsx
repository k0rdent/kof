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
import {
  TimePeriod,
  useTimePeriod,
} from "@/providers/collectors_metrics/TimePeriodState";
import StatRowWithTrend from "./StatRowWithTrend";
import StatRow from "./StatRow";
import { Metric, MetricValue, Pod } from "../pages/collectorPage/models";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "../generated/ui/collapsible";

export interface CustomRowProps {
  rawValue: number;
  formattedValue: string;
  title: string;
}

export interface MetricRow {
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
  rows: MetricRow[];
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
}: MetricsCardProps): JSX.Element => (
  <Card>
    <CardHeader>
      <div className="flex items-center space-x-2 pb-2">
        <Icon className="h-5 w-5" />
        <CardTitle>{title}</CardTitle>
      </div>
      {description && <CardDescription>{description}</CardDescription>}
    </CardHeader>
    <CardContent className="space-y-4">
      {rows.map((row) => (
        <MetricRowComponent key={row.title} row={row} state={state} />
      ))}
    </CardContent>
  </Card>
);

type MetricRowComponentProps = {
  row: MetricRow;
  state: UseBoundStore<StoreApi<DefaultProviderState>>;
};

const MetricRowComponent = ({
  row,
  state,
}: MetricRowComponentProps): JSX.Element => {
  const { selectedPod: pod } = state();
  const { timePeriod } = useTimePeriod();

  if (!pod) return <></>;

  if (row.metricFetchFn) {
    const rawValue = row.metricFetchFn(pod);
    return renderRow(row, rawValue);
  }

  if (!row.metricName) {
    return renderRow(row, 0);
  }

  const metric = pod.getMetric(row.metricName);
  if (!metric) return <></>;

  if (row.customRow) {
    return renderRow(row, metric.totalValue);
  }

  const totalValue = metric.totalValue;
  const formattedValue = row.metricFormat
    ? row.metricFormat(totalValue)
    : totalValue.toString();

  return (
    <MetricCollapsible
      row={row}
      metric={metric}
      formattedValue={formattedValue}
      timePeriod={timePeriod}
    />
  );
};

function renderRow(row: MetricRow, rawValue: number): JSX.Element {
  const formattedValue = row.metricFormat
    ? row.metricFormat(rawValue)
    : rawValue.toString();

  return row.customRow ? (
    row.customRow({ rawValue, formattedValue, title: row.title })
  ) : (
    <StatRow hint={row.hint} text={row.title} value={formattedValue} />
  );
}

type MetricCollapsibleProps = {
  row: MetricRow;
  metric: Metric;
  formattedValue: string;
  timePeriod: TimePeriod;
};

const MetricCollapsible = ({
  row,
  metric,
  formattedValue,
  timePeriod,
}: MetricCollapsibleProps): JSX.Element => {
  const shouldRenderLabels =
    metric.metricValues.filter((value) => value.labelsCount >= 1).length >= 1;

  const cursorStyle: string = `${shouldRenderLabels ? "cursor-pointer" : ""}`;
  return (
    <Collapsible>
      <CollapsibleTrigger className={`w-full mb-2`}>
        {row.enableTrendSystem ? (
          <StatRowWithTrend
            containerStyle={cursorStyle}
            textStyles={cursorStyle}
            key={row.title}
            hint={row.hint}
            text={row.title}
            value={formattedValue}
            trend={metric.getTrend(timePeriod)}
            isPositiveTrend={row.isPositiveTrend}
          />
        ) : (
          <StatRow
            containerStyle={cursorStyle}
            textStyles={cursorStyle}
            key={row.title}
            hint={row.hint}
            text={row.title}
            value={formattedValue}
          />
        )}
      </CollapsibleTrigger>

      {shouldRenderLabels && (
        <CollapsibleContent>
          <div className="flex flex-col bg-muted/40 rounded p-3 space-y-2">
            {metric.metricValues.map((label, idx) => (
              <LabelsRows
                key={idx}
                metricValue={label}
                row={row}
                timePeriod={timePeriod}
              />
            ))}
          </div>
        </CollapsibleContent>
      )}
    </Collapsible>
  );
};

type LabelsRowsProps = {
  metricValue: MetricValue;
  row: MetricRow;
  timePeriod: TimePeriod;
};

const LabelsRows = ({
  metricValue,
  row,
  timePeriod,
}: LabelsRowsProps): JSX.Element => {
  const entries = Object.entries(metricValue.labels);
  const text = entries.map(([key, value]) => `${key}: ${value}`).join("\n");

  const value = row.metricFormat
    ? row.metricFormat(metricValue.numValue)
    : metricValue.numValue.toString();

  return row.enableTrendSystem ? (
    <StatRowWithTrend
      containerStyle="mb-4"
      text={text}
      value={value}
      trend={metricValue.getTrend(timePeriod)}
      isPositiveTrend={row.isPositiveTrend}
    />
  ) : (
    <StatRow containerStyle="mb-4" text={text} value={value} />
  );
};
