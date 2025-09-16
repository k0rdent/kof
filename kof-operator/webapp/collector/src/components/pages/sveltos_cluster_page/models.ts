import { DefaultStatus } from "@/models/DefaultCondition";
import { K8sObject, K8sObjectData } from "@/models/k8sObject";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { Condition, Metadata } from "@/models/ObjectMeta";

export class SveltosClusterSet extends K8sObjectSet<SveltosCluster> {
  protected createK8sObject(
    data: K8sObjectData<SveltosClusterSpecData, SveltosClusterStatusData>,
  ): SveltosCluster {
    return new SveltosCluster(data);
  }
}

export class SveltosCluster extends K8sObject<
  SveltosClusterSpec,
  SveltosClusterStatus,
  SveltosClusterSpecData,
  SveltosClusterStatusData
> {
  public get isHealthy(): boolean {
    return this.status.isHealthy;
  }

  protected createSpec(raw: SveltosClusterSpecData): SveltosClusterSpec {
    return new SveltosClusterSpec(raw);
  }

  protected createStatus(raw: SveltosClusterStatusData): SveltosClusterStatus {
    return new SveltosClusterStatus(raw);
  }
}

export class SveltosClusterSpec {
  private _kubeconfigName: string;
  private _kubeconfigKeyName: string;
  private _paused: boolean;
  private _pullMode: boolean;
  private _consecutiveFailureThreshold: number;

  constructor(data: SveltosClusterSpecData) {
    this._kubeconfigName = data.kubeconfigName;
    this._kubeconfigKeyName = data.kubeconfigKeyName;
    this._paused = data.paused;
    this._pullMode = data.PullMode;
    this._consecutiveFailureThreshold = data.ConsecutiveFailureThreshold;
  }

  public get kubeconfigName(): string {
    return this._kubeconfigName;
  }

  public get kubeconfigKeyName(): string {
    return this._kubeconfigKeyName;
  }

  public get paused(): boolean {
    return this._paused;
  }

  public get pullMode(): boolean {
    return this._pullMode;
  }

  public get consecutiveFailureThreshold(): number {
    return this._consecutiveFailureThreshold;
  }
}

export class SveltosClusterStatus implements DefaultStatus {
  private _connectionStatus: ConnectionStatus;
  private _failureMessage?: string;
  private _connectionFailures: number;
  private _version: string;
  private _ready: boolean;

  public conditions: Condition[] = [];

  constructor(data: SveltosClusterStatusData) {
    this._connectionStatus = data.connectionStatus;
    this._failureMessage = data.failureMessage;
    this._connectionFailures = data.connectionFailures;
    this._version = data.version;
    this._ready = data.ready;

    this.conditions.push({
      name: "Connection",
      status: data.connectionStatus,
      isHealthy: data.connectionStatus === "Healthy",
      message: data.failureMessage,
    });
  }

  public get isHealthy(): boolean {
    return this._connectionStatus === "Healthy";
  }

  public get connectionStatus(): ConnectionStatus {
    return this._connectionStatus;
  }

  public get failureMessage(): string | undefined {
    return this._failureMessage;
  }

  public get connectionFailures(): number {
    return this._connectionFailures;
  }

  public get version(): string {
    return this._version;
  }

  public get ready(): boolean {
    return this._ready;
  }
}

export interface SveltosClusterData {
  spec: SveltosClusterSpecData;
  status: SveltosClusterStatusData;
  metadata: Metadata;
}

interface SveltosClusterSpecData {
  kubeconfigName: string;
  kubeconfigKeyName: string;
  paused: boolean;
  PullMode: boolean;
  ConsecutiveFailureThreshold: number;
}

interface SveltosClusterStatusData {
  version: string;
  ready: boolean;
  connectionStatus: ConnectionStatus;
  failureMessage?: string;
  connectionFailures: number;
}

type ConnectionStatus = "Healthy" | "Down";
