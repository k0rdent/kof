import { METRICS } from "@/constants/metrics.constants";
import {
  MetricsRecordsService,
  Trend,
} from "@/providers/collectors_metrics/CollectorsMetricsRecordManager";
import { TimePeriod } from "@/providers/collectors_metrics/TimePeriodState";

export class ClustersSet {
  private _clusters: Record<string, Cluster> = {};
  private _clustersMap: Record<string, ClusterData> = {};

  constructor(clustersMap: Record<string, ClusterData>) {
    this._clustersMap = clustersMap;
    Object.entries(clustersMap).forEach(([key, value]) => {
      this._clusters[key] = new Cluster(
        key,
        value.customResource,
        value.message,
        value.type,
      );
    });
  }

  public get clusters(): Cluster[] {
    return Object.values(this._clusters);
  }

  public get clusterNames(): string[] {
    return Object.keys(this._clusters);
  }

  public get clustersMap(): Record<string, ClusterData> {
    return this._clustersMap;
  }

  public getCluster(name: string): Cluster | undefined {
    return this._clusters[name];
  }

  public toClusterMap(): Record<string, CustomResourcesMap> {
    return Object.fromEntries(
      this.clusters.map((cluster) => [cluster.name, cluster.getCustomResourceMap()]),
    );
  }
}

export class Cluster {
  private _name: string;
  private _customResource: Record<string, CustomResource> = {};
  private _message?: string | undefined;
  private _error?: string | undefined;
  private _messageType?: MessageType | undefined;

  constructor(
    name: string,
    customResource: CustomResources | null,
    message?: string,
    messageType?: MessageType,
  ) {
    this._name = name;
    this._message = message;
    this._messageType = messageType;

    if (!customResource) {
      return;
    }

    this._customResource = Object.fromEntries(
      Object.entries(customResource).map(([key, value]) => [
        key,
        new CustomResource(this._name, key, value.pods, value.message, value.type),
      ]),
    );
  }

  public get name(): string {
    return this._name;
  }

  public get message(): string | undefined {
    return this._message;
  }

  public get error(): string | undefined {
    return this._error;
  }

  public get customResource(): CustomResource[] {
    return Object.values(this._customResource);
  }

  public get customResourceNames(): string[] {
    return Object.keys(this._customResource);
  }

  public get messageType(): MessageType | undefined {
    return this._messageType;
  }

  public get totalPodCount(): number {
    return this.customResource.reduce((acc, cr) => acc + cr.pods.length, 0);
  }

  public get healthyPodCount(): number {
    return this.customResource.reduce((acc, cr) => acc + cr.healthyPodCount, 0);
  }

  public get unhealthyPodCount(): number {
    return this.customResource.reduce((acc, cr) => acc + cr.unhealthyPodCount, 0);
  }

  public getPod(podName: string): Pod | undefined {
    let pod: Pod | undefined;
    for (const cr of this.customResource) {
      pod = cr.getPod(podName);
      if (pod) break;
    }
    return pod;
  }

  public getCustomResourceMap(): CustomResourcesMap {
    const customResourceMap: CustomResourcesMap = {};
    this.customResource.forEach((cr) => {
      customResourceMap[cr.name] = cr.getPodsMap();
    });
    return customResourceMap;
  }

  public getCustomResource(podName: string): CustomResource | undefined {
    return this._customResource[podName];
  }
}

export class CustomResource {
  private _name: string;
  private _clusterName: string;
  private _pods: Record<string, Pod> = {};
  private _message?: string | undefined;
  private _messageType?: MessageType | undefined;

  constructor(
    clusterName: string,
    name: string,
    pods: Pods | null,
    message?: string,
    messageType?: MessageType,
  ) {
    this._name = name;
    this._message = message;
    this._messageType = messageType;
    this._clusterName = clusterName;

    if (!pods) {
      return;
    }

    this._pods = Object.fromEntries(
      Object.entries(pods).map(([key, value]) => [
        key,
        new Pod(key, clusterName, value.metrics, value.message, value.type),
      ]),
    );
  }

  public get name(): string {
    return this._name;
  }

  public get message(): string | undefined {
    return this._message;
  }

  public get messageType(): MessageType | undefined {
    return this._messageType;
  }

  public get clusterName(): string {
    return this._clusterName;
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
  private _messageType?: MessageType;
  private _message?: string;

  constructor(
    name: string,
    clusterName: string,
    metrics: MetricsMap,
    message?: string,
    messageType?: MessageType,
  ) {
    this._clusterName = clusterName;
    this._name = name;
    this._message = message;
    this._messageType = messageType;

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

  public get message(): string | undefined {
    return this._message;
  }

  public get messageType(): MessageType | undefined {
    return this._messageType;
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
    metricValues: MetricValueJson[],
  ) {
    this._name = name;
    this._clusterName = clusterName;
    this._podName = podName;
    this._metricValues = metricValues.map(
      (l) => new MetricValue({ ...l }, clusterName, podName, name),
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
      timePeriod,
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
    metricName: string,
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
      this.id,
    );
  }
}

export type PodsMap = Record<string, MetricsMap>;
type MetricsMap = Record<string, MetricValueJson[]>;
type Labels = Record<string, string>;
type MetricValueJson = {
  value: number | string;
  labels?: Labels;
};

type CustomResourcesMap = Record<string, PodsMap>;
type CustomResources = Record<string, CustomResourceData>;
type Pods = Record<string, PodData>;

export type MessageType = "info" | "warning" | "error";

type BaseResourceStatus = {
  name?: string;
  message?: string;
  type?: MessageType;
};

type CustomResourceData = BaseResourceStatus & {
  pods: Pods;
};

type PodData = BaseResourceStatus & {
  metrics: MetricsMap;
};

export type ClusterData = BaseResourceStatus & {
  customResource: CustomResources;
};

export interface StatusMessage {
  cluster: string;
  customResource?: string;
  pod?: string;
  type: MessageType;
  message: string;
  details?: string;
}

export interface MetricData {
  cluster: string;
  customResource?: string;
  pod?: string;
  name: string;
  value?: MetricValueJson;
  error?: string;
}

export interface ResourceMessage {
  status?: StatusMessage;
  metrics?: MetricData;
}
