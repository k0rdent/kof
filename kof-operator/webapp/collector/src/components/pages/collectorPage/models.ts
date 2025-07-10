import { METRICS } from "@/constants/metrics.constants";

export class CollectorMetrics {
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
}

export class Cluster {
  private _name: string;
  private _pods: Record<string, Pod> = {};

  constructor(name: string, pods: PodsMap) {
    this._name = name;
    Object.entries(pods).forEach(([key, value]) => {
      this._pods[key] = new Pod(key, value);
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

  public getPod(podName: string): Pod {
    return this._pods[podName];
  }
}

export class Pod {
  private _name: string;
  private _metrics: Metrics;

  constructor(name: string, metrics: Metrics) {
    this._metrics = metrics;
    this._name = name;
  }

  public get name(): string {
    return this._name;
  }

  public get isHealthy(): boolean {
    return (
      this.getStringMetric(METRICS.OTELCOL_CONDITION_READY_HEALTHY) == "healthy"
    );
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

type PodsMap = Record<string, Metrics>;
type Metrics = Record<string, number | string>;
