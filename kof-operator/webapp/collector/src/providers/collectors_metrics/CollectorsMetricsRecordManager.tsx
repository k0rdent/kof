import {
  CollectorMetrics,
  Pod,
  PodsMap,
} from "@/components/pages/collectorPage/models";
import { TimePeriod } from "./TimePeriodHook";
import { bytesToUnits, formatNumber } from "@/utils/formatter";
import Dexie, { type EntityTable } from "dexie";

export type Metric = {
  timestamp: number;
  name: string;
  value: number;
};

type MetricsRecord = {
  timestamp: number;
  metrics: CollectorMetrics;
};

type MetricsHist = {
  timestamp: number;
  record: Record<string, PodsMap>;
};

export interface Trend {
  message: string;
  isTrending: boolean;
  changesWithTime: number;
}

export class CollectorMetricsRecordsManager {
  private _records: MetricsRecord[] = [];
  private _db: MetricsDatabase = new MetricsDatabase();

  constructor() {
    this.get();
  }

  private async get(): Promise<void> {
    const records: MetricsRecord[] = [];
    const timeNow = new Date().getTime();
    const timeThen = timeNow - 60 * 60 * 1000;
    const values: MetricsHist[] = await this._db.getRecords(timeThen, timeNow);

    values.forEach((value) => {
      const metrics = new CollectorMetrics(value.record);

      records.push({
        metrics,
        timestamp: value.timestamp,
      });
    });

    this._records = records;
  }

  public add(metrics: CollectorMetrics): void {
    const record: MetricsRecord = { timestamp: new Date().getTime(), metrics };
    this._records.push(record);
    this._db.addRecord(record.timestamp, record.metrics.toClusterMap());
  }

  public getMetricHistory(collector: Pod, metricName: string): Metric[] {
    const metricHistory: Metric[] = [];
    this._records.forEach((record) => {
      const cluster = record.metrics.getCluster(collector.clusterName);
      if (!cluster) {
        return;
      }

      const pod = cluster.getPod(collector.name);
      if (!pod) {
        return;
      }

      metricHistory.push({
        name: metricName,
        value: pod.getMetric(metricName),
        timestamp: record.timestamp,
      });
    });

    return metricHistory;
  }

  public calculateAverage(timePeriod: TimePeriod, metrics: Metric[]): Trend {
    const timeAgo = new Date().getTime() - timePeriod.value * 1000;
    const timeRange: Metric[] = [];

    metrics.forEach((m) => {
      const time = new Date(m.timestamp).getTime();
      if (timeAgo < time) {
        timeRange.push(m);
      }
    });

    if (timeRange.length == 1) {
      return {
        message: `In Average ${bytesToUnits(timeRange[0].value)}`,
        isTrending: true,
        changesWithTime: 0,
      };
    }
    timeRange.sort((a, b) => a.timestamp - b.timestamp);
    
    const sum = timeRange
      .map((metric) => metric.value)
      .reduce((a, b) => a + b, 0);
      console.log(sum)
    const average = sum / timeRange.length || 0;
    const formattedChangesWithTime = bytesToUnits(average);

    return {
      message: `In Average ${formattedChangesWithTime}`,
      isTrending: true,
      changesWithTime: 0,
    };
  }

  public calculateTrend(timePeriod: TimePeriod, metrics: Metric[]): Trend {
    const timeAgo = new Date().getTime() - timePeriod.value * 1000;
    const timeRange: Metric[] = [];

    metrics.forEach((m) => {
      const time = new Date(m.timestamp).getTime();
      if (timeAgo < time) {
        timeRange.push(m);
      }
    });

    if (timeRange.length <= 1) {
      return {
        message: `None in ${timePeriod.text}`,
        isTrending: false,
        changesWithTime: 0,
      };
    }

    timeRange.sort((a, b) => a.timestamp - b.timestamp);
    const first = timeRange[0].value;
    const last = timeRange[timeRange.length - 1].value;

    const isTrending = first < last;
    const changesWithTime = last - first;
    const formattedChangesWithTime = formatNumber(changesWithTime);
    const message = isTrending
      ? `${formattedChangesWithTime} in ${timePeriod.text}`
      : `None in ${timePeriod.text}`;

    return {
      message,
      isTrending,
      changesWithTime,
    };
  }
}

class MetricsDatabase {
  private _db: Dexie & {
    metrics: EntityTable<MetricsHist, "timestamp">;
  };

  constructor() {
    this._db = new Dexie("MetricsDatabase") as Dexie & {
      metrics: EntityTable<MetricsHist, "timestamp">;
    };

    this._db.version(1).stores({
      metrics: "++timestamp, record",
    });
  }

  public async getRecords(
    minTime: number,
    maxTime: number
  ): Promise<MetricsHist[]> {
    return await this._db.metrics
      .where("timestamp")
      .between(minTime, maxTime)
      .toArray();
  }

  public async addRecord(
    timestamp: number,
    record: Record<string, PodsMap>
  ): Promise<void> {
    await this._db.metrics.add({
      timestamp,
      record,
    });
  }
}
