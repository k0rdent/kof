import { METRICS } from "@/constants/metrics.constants";

export class CollectorMetricsSet {
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

  public getCluster(name: string): Cluster {
    return this._clusters[name];
  }

  public toClusterMap(): Record<string, PodsMap> {
    const clustersMap: Record<string, PodsMap> = {};
    this.clusters.forEach((cluster) => {
      clustersMap[cluster.name] = cluster.getPodsMap();
    });
    return clustersMap;
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
    
    Object.entries(pods).forEach(([key, value]) => {
      this._pods[key] = new Pod(key, name, value);
    });
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

  public getPod(podName: string): Pod {
    return this._pods[podName];
  }
}

export class Pod {
  private _clusterName: string;
  private _name: string;
  private _metrics: MetricsMap;

  constructor(name: string, clusterName: string, metrics: MetricsMap) {
    this._clusterName = clusterName;
    this._metrics = metrics;
    this._name = name;
  }

  public get name(): string {
    return this._name;
  }

  public get clusterName(): string {
    return this._clusterName;
  }

  public get isHealthy(): boolean {
    return (
      this.getStringMetric(METRICS.OTELCOL_CONDITION_READY_HEALTHY) == "healthy"
    );
  }

  public getMetrics(): MetricsMap {
    return this._metrics;
  }

  public getMetric(metricName: string): number {
    const value = this._metrics[metricName];
    return typeof value === "number" ? value : 0;
  }

  public getStringMetric(metricName: string): string {
    const value = this._metrics[metricName];
    return typeof value === "string" ? value : "";
  }
}

export type PodsMap = Record<string, MetricsMap>;
type MetricsMap = Record<string, number | string>;
