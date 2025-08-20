import { METRICS } from "@/constants/metrics.constants";
import {
  MetricsRecordsService,
  Trend,
} from "@/providers/collectors_metrics/CollectorsMetricsRecordManager";
import { TimePeriod } from "@/providers/collectors_metrics/TimePeriodState";

export class ClustersSet {
  private _clusters: Record<string, Cluster> = {};

  constructor(clustersMap: Record<string, PodsMap>) {
    Object.entries(clustersMap).forEach(([key, value]) => {
      this._clusters[key] = new Cluster(key, value);
    });
  }

  public get clusters(): Cluster[] {
    return Object.values(this._clusters);
  }

  public get clusterNames(): string[] {
    return Object.keys(this._clusters);
  }

  public getCluster(name: string): Cluster | undefined {
    return this._clusters[name];
  }

  public toClusterMap(): Record<string, PodsMap> {
    return Object.fromEntries(
      this.clusters.map((cluster) => [cluster.name, cluster.getPodsMap()])
    );
  }
}

export class Cluster {
  private _name: string;
  private _pods: Record<string, Pod> = {};

  constructor(name: string, pods: PodsMap | null) {
    this._name = name;
    if (!pods) {
      return;
    }

    this._pods = Object.fromEntries(
      Object.entries(pods).map(([key, value]) => [
        key,
        new Pod(key, name, value),
      ])
    );
  }

  public get name(): string {
    return this._name;
  }

  public get pods(): Pod[] {
    return Object.values(this._pods);
  }

  public get podNames(): string[] {
    return Object.keys(this._pods);
  }

  public get healthyPodCount(): number {
    return this.pods.filter((pod) => pod.isHealthy).length;
  }

  public get unhealthyPodCount(): number {
    return this.pods.filter((pod) => !pod.isHealthy).length;
  }

  public getPodsMap(): PodsMap {
    const podsMap: PodsMap = {};
    this.pods.forEach((pod) => {
      podsMap[pod.name] = pod.getMetrics();
    });
    return podsMap;
  }

  public getPod(podName: string): Pod | undefined {
    return this._pods[podName];
  }
}

export class Pod {
  private _clusterName: string;
  private _name: string;
  private _metrics: Record<string, Metric> = {};

  constructor(
    name: string,
    clusterName: string,
    metrics: MetricsMap
  ) {
    this._clusterName = clusterName;
    this._name = name;

    Object.entries(metrics).forEach(([key, value]) => {
      if (!value) return;
      const mValue = value.map((v) => ({ ...v }));
      this._metrics[key] = new Metric(key, name, clusterName, mValue);
    });
  }

  public get name(): string {
    return this._name;
  }

  public get clusterName(): string {
    return this._clusterName;
  }

  public get isHealthy(): boolean {
    const metric = this.getMetric(METRICS.CONDITION_READY_HEALTHY.name);
    return metric?.metricValues[0]?.strValue === "healthy";
  }

  public get metrics(): Metric[] {
    return Object.values(this._metrics);
  }

  public get metricsMap(): Record<string, Metric> {
    return this._metrics;
  }

  public getMetrics(): MetricsMap {
    const metricsMap: MetricsMap = {};
    Object.entries(this._metrics).forEach(([key, value]) => {
      metricsMap[key] = value.metricValuesJson;
    });
    return metricsMap;
  }

  public getMetric(metricName: string): Metric | undefined {
    return this._metrics[metricName];
  }
}

export class Metric {
  private _name: string;
  private _metricValues: MetricValue[] = [];
  private _podName: string;
  private _clusterName: string;

  constructor(
    name: string,
    podName: string,
    clusterName: string,
    metricValues: MetricValueJson[]
  ) {
    this._name = name;
    this._clusterName = clusterName;
    this._podName = podName;
    this._metricValues = metricValues.map(
      (l) => new MetricValue({ ...l }, clusterName, podName, name)
    );
  }

  public get name(): string {
    return this._name;
  }

  public get podName(): string {
    return this._podName;
  }

  public get clusterName(): string {
    return this._clusterName;
  }

  public get totalValue(): number {
    return this._metricValues.reduce((pv, cv) => pv + cv.numValue, 0) ?? 0;
  }

  public get metricValues(): MetricValue[] {
    return this._metricValues;
  }

  public get metricValuesJson(): MetricValueJson[] {
    return this._metricValues.map((l) => ({
      value: l.value,
      labels: l.labels,
    }));
  }

  public getTrend(timePeriod: TimePeriod): Trend {
    return MetricsRecordsService.getTrend(
      this._clusterName,
      this._podName,
      this._name,
      timePeriod
    );
  }

  public getLabelsById(id: string): MetricValue | undefined {
    return this._metricValues.find((labels) => labels.id === id);
  }
}

export class MetricValue {
  private _value: string | number;
  private _labels: Labels;
  private _clusterName: string;
  private _podName: string;
  private _metricName: string;

  constructor(
    metricValueJson: MetricValueJson,
    clusterName: string,
    podName: string,
    metricName: string
  ) {
    this._labels = metricValueJson.labels ? metricValueJson.labels : {};
    this._value = metricValueJson.value;
    this._clusterName = clusterName;
    this._podName = podName;
    this._metricName = metricName;
  }

  public get labels(): Readonly<Labels> {
    return this._labels;
  }

  public get labelsCount(): number {
    return Object.keys(this._labels).length;
  }

  public get value(): number | string {
    return this._value;
  }

  public get strValue(): string {
    return String(this._value);
  }

  public get numValue(): number {
    const num = Number(this._value);
    return Number.isFinite(num) ? num : 0;
  }

  public get id(): string {
    return Object.entries(this._labels)
      .map(([key, val]) => `${key}=${val}`)
      .sort()
      .join("#");
  }

  public getTrend(timePeriod: TimePeriod): Trend {
    return MetricsRecordsService.getTrend(
      this._clusterName,
      this._podName,
      this._metricName,
      timePeriod,
      this.id
    );
  }
}

export type PodsMap = Record<string, MetricsMap>;
type MetricsMap = Record<string, MetricValueJson[]>;
type Labels = Record<string, string>;
type MetricValueJson = {
  value: number | string;
  labels: Labels;
};
