import { Pod } from "@/components/pages/collectorPage/models";
import {
  CollectorMetricsRecordsManager,
  Trend,
} from "@/providers/collectors_metrics/CollectorsMetricsRecordManager";
import { TimePeriod } from "@/providers/collectors_metrics/TimePeriodState";

export function getMetricTrendData(
  metricName: string,
  history: CollectorMetricsRecordsManager,
  collector: Pod,
  timePeriod: TimePeriod
): {
  metricValue: number;
  metricTrend: Trend;
} {
  const metricValue = collector.getMetric(metricName);
  const metricHistory = history.getMetricHistory(collector, metricName);
  const metricTrend = history.getMetricTrend(timePeriod, metricHistory);

  return {
    metricValue,
    metricTrend,
  };
}

export function getAverageValue(
  metricName: string,
  history: CollectorMetricsRecordsManager,
  collector: Pod,
  timePeriod: TimePeriod
): number {
  const metricHistory = history.getMetricHistory(collector, metricName);
  return history.getAverageMetricValue(timePeriod, metricHistory);
}
