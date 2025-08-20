import { Pod } from "@/components/pages/collectorPage/models";
import {
  MetricsRecordsManager,
  Trend,
} from "@/providers/collectors_metrics/CollectorsMetricsRecordManager";
import { TimePeriod } from "@/providers/collectors_metrics/TimePeriodState";

export function getMetricTrendData(
  metricName: string,
  history: MetricsRecordsManager,
  collector: Pod,
  timePeriod: TimePeriod
): {
  metricValue: number;
  metricTrend: Trend;
} {
  const metricValue = collector.getMetric(metricName)?.totalValue ?? 0;
  const metricHistory = history.getMetricHistory(collector, metricName);
  const metricTrend = history.getMetricTrend(timePeriod, metricHistory);

  return {
    metricValue,
    metricTrend,
  };
}

export function getAverageValue(
  metricName: string,
  history: MetricsRecordsManager,
  collector: Pod,
  timePeriod: TimePeriod
): number {
  const metricHistory = history.getMetricHistory(collector, metricName);
  return history.getAverageMetricValue(timePeriod, metricHistory);
}
