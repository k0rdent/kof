import { DefaultCondition, DefaultStatus } from "@/models/DefaultCondition";
import { K8sObjectData, K8sObject } from "@/models/k8sObject";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { ClusterConditionData, Metadata } from "@/models/ObjectMeta";

const CLUSTER_ROLE_LABEL = "k0rdent.mirantis.com/kof-cluster-role";

export type ClusterRole = "child" | "regional";

export class ClusterDeploymentSet extends K8sObjectSet<ClusterDeployment> {
  protected createK8sObject(
    path: string,
    data: K8sObjectData<ClusterSpecData, ClusterStatusData>
  ): ClusterDeployment {
    return new ClusterDeployment(path, data);
  }
}

export class ClusterDeployment extends K8sObject<
  ClusterSpec,
  ClusterStatus,
  ClusterSpecData,
  ClusterStatusData
> {
  public get isHealthy(): boolean {
    return !this.status.conditions.find((c) => !c.isHealthy);
  }

  protected createSpec(data: ClusterSpecData): ClusterSpec {
    return new ClusterSpec(data);
  }

  protected createStatus(data: ClusterStatusData): ClusterStatus {
    return new ClusterStatus(data);
  }

  public get role(): ClusterRole | undefined {
    const role: string = this.metadata.labels[CLUSTER_ROLE_LABEL];
    if (role === "child" || role === "regional") {
      return role;
    }
    return undefined;
  }

  public get deletionTime(): Date | null {
    return this.metadata.deletionDate ? new Date(this.metadata.deletionDate) : null;
  }

  public get totalNodes(): number {
    return (
      (this.spec.config.controlPlaneNumber ?? 0) + (this.spec.config.workersNumber ?? 0)
    );
  }
}

export class ClusterSpec {
  private _config: ClusterConfig;
  private _template: string;
  private _credential: string;
  private _ipamClaim?: Record<string, unknown> | undefined;
  private _serviceSpec?: ServiceSpecData | undefined;
  private _propagateCredentials?: boolean | undefined;

  constructor(data: ClusterSpecData) {
    this._template = data.template;
    this._credential = data.credential;
    this._ipamClaim = data.ipamClaim;
    this._serviceSpec = data.serviceSpec;
    this._propagateCredentials = data.propagateCredentials;
    this._config = new ClusterConfig(data.config);
  }

  public get config(): ClusterConfig {
    return this._config;
  }

  public get template(): string {
    return this._template;
  }

  public get credential(): string {
    return this._credential;
  }

  public get ipamClaim(): Record<string, unknown> | undefined {
    return this._ipamClaim;
  }

  public get serviceSpec(): ServiceSpecData | undefined {
    return this._serviceSpec;
  }

  public get propagateCredentials(): boolean | undefined {
    return this._propagateCredentials;
  }

  public get provider(): string {
    return this._template.split("-")[0];
  }
}

export class ClusterConfig {
  private _clusterAnnotation?: Record<string, string>;
  private _clusterIdentity: ClusterIdentityData;
  private _controlPlane?: ControlPlaneData;
  private _controlPlaneNumber?: number;
  private _region?: string;
  private _worker?: WorkerSpecData;
  private _workersNumber?: number;

  constructor(data: ClusterConfigData) {
    this._clusterAnnotation = data.clusterAnnotations;
    this._clusterIdentity = data.clusterIdentity;
    this._controlPlane = data.controlPlane;
    this._controlPlaneNumber = data.controlPlaneNumber;
    this._region = data.region;
    this._worker = data.worker;
    this._workersNumber = data.workersNumber;
  }

  public get clusterAnnotations(): Record<string, string> | undefined {
    return this._clusterAnnotation;
  }

  public get clusterAnnotationsArray(): [string, string][] {
    if (!this._clusterAnnotation) return [];
    return Object.entries(this._clusterAnnotation);
  }

  public get clusterIdentity(): ClusterIdentityData {
    return this._clusterIdentity;
  }

  public get controlPlane(): ControlPlaneData | undefined {
    return this._controlPlane;
  }

  public get controlPlaneNumber(): number | undefined {
    return this._controlPlaneNumber;
  }

  public get region(): string | undefined {
    return this._region;
  }

  public get worker(): WorkerSpecData | undefined {
    return this._worker;
  }

  public get workersNumber(): number | undefined {
    return this._workersNumber;
  }
}

export class ClusterStatus implements DefaultStatus {
  private _conditions: DefaultCondition[];
  private _observedGeneration: number;
  private _healthyConditions: number;
  private _unhealthyConditions: number;

  constructor(data: ClusterStatusData) {
    this._conditions = data.conditions.map((c) => new DefaultCondition(c));
    this._observedGeneration = data.observedGeneration;
    this._healthyConditions = this._conditions.filter((c) => c.isHealthy).length;
    this._unhealthyConditions = this._conditions.length - this._healthyConditions;
  }

  public get conditions(): DefaultCondition[] {
    return this._conditions;
  }

  public get healthyConditions(): number {
    return this._healthyConditions;
  }

  public get unhealthyConditions(): number {
    return this._unhealthyConditions;
  }

  public get observedGeneration(): number {
    return this._observedGeneration;
  }
}

export interface ClusterDeploymentData {
  spec: ClusterSpecData;
  status: ClusterStatusData;
  metadata: Metadata;
}

export interface ClusterSpecData {
  config: ClusterConfigData;
  template: string;
  credential: string;
  ipamClaim?: Record<string, unknown>;
  serviceSpec?: ServiceSpecData;
  propagateCredentials?: boolean;
}

export interface ClusterConfigData {
  clusterAnnotations?: Record<string, string>;
  clusterIdentity: ClusterIdentityData;
  region: string;
  controlPlane?: ControlPlaneData;
  controlPlaneNumber?: number;
  worker?: WorkerSpecData;
  workersNumber?: number;
}

interface ClusterIdentityData {
  name: string;
  namespace: string;
}

interface ControlPlaneData {
  instanceType: string;
}

interface WorkerSpecData {
  instanceType: string;
}

interface ServiceSpecData {
  syncMode: string;
  priority: number;
}

interface ClusterStatusData {
  conditions: ClusterConditionData[];
  observedGeneration: number;
}
