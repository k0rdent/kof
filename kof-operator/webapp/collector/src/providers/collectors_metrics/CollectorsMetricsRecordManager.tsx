import { ClustersSet, Pod } from "@/components/pages/collectorPage/models";
import { TimePeriod } from "./TimePeriodState";
import { formatNumber } from "@/utils/formatter";
import { MetricsDatabase } from "@/database/MetricsDatabase";

export type MetricHistoryEntry = {
  timestamp: number;
  name: string;
  value: number;
};

type MetricsRecord = {
  timestamp: number;
  metrics: ClustersSet;
};

export interface Trend {
  message: string;
  isTrending: boolean;
  changesWithTime: number;
}

export class MetricsRecordsManager {
  private _cachedRecords: MetricsRecord[] = [];
  private _db: MetricsDatabase = new MetricsDatabase();

  constructor() {
    this.init();
  }

  private async init(): Promise<void> {
    const now = new Date().getTime();
    const oneHourAgo = now - 60 * 60 * 1000;

    const fetchedRecords = await this._db.getRecords(oneHourAgo, now);
    await this._db.deleteOldRecords(oneHourAgo);

    this._cachedRecords = fetchedRecords.map((item) => ({
      timestamp: item.timestamp,
      metrics: new ClustersSet(item.record),
    }));
  }

  public async add(metrics: ClustersSet): Promise<void> {
    const timestamp = Date.now();
    const record: MetricsRecord = { timestamp, metrics };
    this._cachedRecords.push(record);
    await this._db.addRecord(record.timestamp, record.metrics.toClusterMap());
  }

  public getTrend(
    clusterName: string,
    podName: string,
    metricName: string,
    timePeriod: TimePeriod,
    labelId?: string
  ): Trend {
    const metricHistory: MetricHistoryEntry[] = [];
    for (const record of this._cachedRecords) {
      const cluster = record.metrics.getCluster(clusterName);
      if (!cluster) continue;

      const pod = cluster.getPod(podName);
      if (!pod) continue;

      const metric = pod.getMetric(metricName);
      if (!metric) continue;

      if (labelId) {
        const label = metric.getLabelsById(labelId);
        if (!label) continue;
        metricHistory.push({
          name: metricName,
          value: label.numValue,
          timestamp: record.timestamp,
        });
      } else {
        metricHistory.push({
          name: metricName,
          value: metric.totalValue,
          timestamp: record.timestamp,
        });
      }
    }

    return this.getMetricTrend(timePeriod, metricHistory);
  }

  public getMetricHistory(pod: Pod, metricName: string): MetricHistoryEntry[] {
    const metricHistory: MetricHistoryEntry[] = [];

    for (const record of this._cachedRecords) {
      const recordCluster = record.metrics.getCluster(pod.clusterName);
      const recordPod = recordCluster?.getPod(pod.name);
      if (!recordPod) {
        continue;
      }

      metricHistory.push({
        name: metricName,
        value: recordPod.getMetric(metricName)?.totalValue ?? 0,
        timestamp: record.timestamp,
      });
    }

    return metricHistory;
  }

  public getAverageMetricValue(
    timePeriod: TimePeriod,
    metrics: MetricHistoryEntry[]
  ): number {
    const recentMetrics = this.filterRecentMetrics(metrics, timePeriod);

    if (recentMetrics.length === 0) return 0;
    if (recentMetrics.length === 1) return recentMetrics[0].value;

    const sum = recentMetrics.reduce((sum, m) => sum + m.value, 0);
    return sum / recentMetrics.length;
  }

  public getMetricTrend(
    timePeriod: TimePeriod,
    metrics: MetricHistoryEntry[]
  ): Trend {
    const recentMetrics = this.filterRecentMetrics(metrics, timePeriod);
    if (recentMetrics.length <= 1) {
      return {
        message: `0 in ${timePeriod.text}`,
        isTrending: false,
        changesWithTime: 0,
      };
    }

    recentMetrics.sort((a, b) => a.timestamp - b.timestamp);
    const first = recentMetrics[0].value;
    const last = recentMetrics[recentMetrics.length - 1].value;

    const isTrending = first < last;
    const changesWithTime = last - first;
    const formattedChangesWithTime = formatNumber(changesWithTime);
    const message = `${formattedChangesWithTime} in ${timePeriod.text}`;

    return {
      message,
      isTrending,
      changesWithTime,
    };
  }

  private filterRecentMetrics(
    metrics: MetricHistoryEntry[],
    timePeriod: TimePeriod
  ): MetricHistoryEntry[] {
    const cutoff = Date.now() - timePeriod.value * 1000;
    return metrics.filter((m) => m.timestamp > cutoff);
  }
}

export const MetricsRecordsService = new MetricsRecordsManager();
