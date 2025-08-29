const CLUSTER_ROLE_LABEL = "k0rdent.mirantis.com/kof-cluster-role";

export type ClusterRole = "child" | "regional";

export class ClusterDeploymentSet {
  private _deploymentsMap: Record<string, ClusterDeployment>;
  private _deploymentsArray: ClusterDeployment[];
  private _healthyCount: number;
  private _unhealthyCount: number;

  constructor(data: Record<string, ClusterDeploymentData>) {
    this._deploymentsMap = Object.entries(data).reduce((acc, [name, value]) => {
      acc[name] = new ClusterDeployment(value);
      return acc;
    }, {} as Record<string, ClusterDeployment>);

    this._deploymentsArray = Object.values(this._deploymentsMap);

    this._healthyCount = this._deploymentsArray.filter(
      (c) => c.isHealthy
    ).length;

    this._unhealthyCount = this._deploymentsArray.length - this._healthyCount;
  }

  public get length(): number {
    return this._deploymentsArray.length;
  }

  public get healthyCount(): number {
    return this._healthyCount;
  }

  public get unhealthyCount(): number {
    return this._unhealthyCount;
  }

  public get isHealthy(): boolean {
    console.log(this._unhealthyCount === 0)
    return this._unhealthyCount === 0;
  }

  public get deployments(): ClusterDeployment[] {
    return this._deploymentsArray;
  }

  public getCluster(name: string): ClusterDeployment | null {
    return this._deploymentsMap[name] ?? null;
  }
}

export class ClusterDeployment {
  private _name: string;
  private _namespace: string;
  private _labels: Record<string, string>;
  private _annotations: Record<string, string>;
  private _spec: ClusterSpec;
  private _status: ClusterStatus;
  private _creation_time: string;
  private _deletion_time?: string;
  private _generation: number;

  constructor(data: ClusterDeploymentData) {
    this._name = data.name;
    this._namespace = data.namespace;
    this._labels = data.labels;
    this._annotations = data.annotations;
    this._creation_time = data.creation_time;
    this._deletion_time = data.deletion_time;
    this._generation = data.generation;
    this._status = new ClusterStatus(data.status);
    this._spec = new ClusterSpec(data.spec);
  }

  public get name(): string {
    return this._name;
  }

  public get namespace(): string {
    return this._namespace;
  }

  public get generation(): number {
    return this._generation;
  }

  public get labels(): Record<string, string> {
    return this._labels;
  }

  public get annotations(): Record<string, string> {
    return this._annotations;
  }

  public get spec(): ClusterSpec {
    return this._spec;
  }

  public get status(): ClusterStatus {
    return this._status;
  }

  public get role(): ClusterRole | undefined {
    const role: string = this._labels[CLUSTER_ROLE_LABEL];
    if (role === "child" || role === "regional") {
      return role;
    }
    return undefined;
  }

  public get isReady(): boolean {
    return this.status.conditions.some(
      (c) => c.type === "Ready" && c.status === "True"
    );
  }

  public get totalStatusCount(): number {
    return this._status.conditions.length;
  }

  public get healthyStatusCount(): number {
    return this._status.conditions.filter((c) => c.status === "True").length;
  }

  public get unhealthyStatusCount(): number {
    return this.totalStatusCount - this.healthyStatusCount;
  }

  public get isHealthy(): boolean {
    return this.healthyStatusCount === this._status.conditions.length;
  }

  public get creationTime(): Date {
    return new Date(this._creation_time);
  }

  public get deletionTime(): Date | null {
    return this._deletion_time ? new Date(this._deletion_time) : null;
  }

  public get totalNodes(): number {
    return (
      this._spec.config.controlPlaneNumber + this._spec.config.workersNumber
    );
  }

  public get ageInSeconds(): number {
    const timeNow: number = Date.now();
    const creationTime: number = this.creationTime.getTime();
    return (timeNow - creationTime) / 1000;
  }
}

export class ClusterSpec implements ClusterSpecData {
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

export class ClusterConfig implements ClusterConfigData {
  private _clusterAnnotation?: Record<string, string>;
  private _clusterIdentity: ClusterIdentityData;
  private _controlPlane: ControlPlaneData;
  private _controlPlaneNumber: number;
  private _region: string;
  private _worker: WorkerSpecData;
  private _workersNumber: number;

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

  public get controlPlane(): ControlPlaneData {
    return this._controlPlane;
  }

  public get controlPlaneNumber(): number {
    return this._controlPlaneNumber;
  }

  public get region(): string {
    return this._region;
  }

  public get worker(): WorkerSpecData {
    return this._worker;
  }

  public get workersNumber(): number {
    return this._workersNumber;
  }
}

export class ClusterStatus implements ClusterStatusData {
  private _conditions: ClusterCondition[];
  private _observedGeneration: number;

  constructor(data: ClusterStatusData) {
    this._conditions = data.conditions.map((c) => new ClusterCondition(c));
    this._observedGeneration = data.observedGeneration;
  }

  public get conditions(): ClusterCondition[] {
    return this._conditions;
  }

  public get observedGeneration(): number {
    return this._observedGeneration;
  }
}

export class ClusterCondition implements ClusterConditionData {
  private _type: string;
  private _status: string;
  private _observedGeneration?: number | undefined;
  private _lastTransitionTime: string;
  private _lastTransitionTimeDate: Date;
  private _reason: string;
  private _message: string;

  constructor(data: ClusterConditionData) {
    this._lastTransitionTime = data.lastTransitionTime;
    this._type = data.type;
    this._status = data.status;
    this._reason = data.reason;
    this._message = data.message;
    this._lastTransitionTimeDate = new Date(data.lastTransitionTime);
  }

  public get type(): string {
    return this._type;
  }

  public get status(): string {
    return this._status;
  }

  public get observedGeneration(): number | undefined {
    return this._observedGeneration;
  }

  public get lastTransitionTime(): string {
    return this._lastTransitionTime;
  }

  public get reason(): string {
    return this._reason;
  }

  public get message(): string {
    return this._message;
  }

  public get lastTransitionTimeDate(): Date {
    return this._lastTransitionTimeDate;
  }

  public get isHealthy(): boolean {
    return this._status === "True";
  }
}

export interface ClusterDeploymentData {
  name: string;
  namespace: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  spec: ClusterSpecData;
  status: ClusterStatusData;
  generation: number;
  creation_time: string;
  deletion_time?: string;
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
  controlPlane: ControlPlaneData;
  controlPlaneNumber: number;
  region: string;
  worker: WorkerSpecData;
  workersNumber: number;
}

export interface ClusterIdentityData {
  name: string;
  namespace: string;
}

export interface ControlPlaneData {
  instanceType: string;
}

export interface WorkerSpecData {
  instanceType: string;
}

export interface ServiceSpecData {
  syncMode: string;
  priority: number;
}

export interface ClusterStatusData {
  conditions: ClusterConditionData[];
  observedGeneration: number;
}

export interface ClusterConditionData {
  type: string;
  status: string;
  observedGeneration?: number;
  lastTransitionTime: string;
  reason: string;
  message: string;
}
