import { DefaultCondition, DefaultStatus } from "@/models/DefaultCondition";
import { K8sObject, K8sObjectData } from "@/models/k8sObject";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { ClusterConditionData, Condition } from "@/models/ObjectMeta";

export class ServiceSetListSet extends K8sObjectSet<ServiceSet> {
  protected createK8sObject(
    path: string,
    data: K8sObjectData<ServiceSetSpecData, ServiceSetStatusData>,
  ): ServiceSet {
    return new ServiceSet(path, data);
  }
}

export class ServiceSet extends K8sObject<
  ServiceSetSpec,
  ServiceSetStatus,
  ServiceSetSpecData,
  ServiceSetStatusData
> {
  public get isHealthy(): boolean {
    const conditionsHealthy = this.status.conditions.every((c) => c.isHealthy);
    const servicesHealthy = this.status.services?.every((s) => s.isHealthy) ?? true;
    return conditionsHealthy && servicesHealthy;
  }

  protected createSpec(data: ServiceSetSpecData): ServiceSetSpec {
    return new ServiceSetSpec(data);
  }

  protected createStatus(data: ServiceSetStatusData): ServiceSetStatus {
    return new ServiceSetStatus(data);
  }
}

export class ServiceSetSpec {
  private _cluster: string;
  private _multiClusterService?: string;
  private _services?: ServiceSetServiceSpec[];

  constructor(data: ServiceSetSpecData) {
    this._cluster = data.cluster;
    this._multiClusterService = data.multiClusterService;
    this._services = data.services?.map((s) => new ServiceSetServiceSpec(s));
  }

  public get cluster(): string {
    return this._cluster;
  }

  public get multiClusterService(): string | undefined {
    return this._multiClusterService;
  }

  public get services(): ServiceSetServiceSpec[] | undefined {
    return this._services;
  }
}

export class ServiceSetStatus implements DefaultStatus {
  private _conditions: DefaultCondition[];
  private _services?: ServiceSetServiceStatus[];
  private _deployed: boolean;
  private _provider: { ready: boolean };

  constructor(data: ServiceSetStatusData) {
    this._deployed = data.deployed;
    this._provider = data.provider;
    this._conditions = data.conditions.map((c) => new DefaultCondition(c));
    this._services = data.services?.map((s) => new ServiceSetServiceStatus(s));
  }

  public get conditions(): Condition[] {
    return this._conditions;
  }

  public get deployed(): boolean {
    return this._deployed;
  }

  public get provider(): { ready: boolean } {
    return this._provider;
  }

  public get services(): ServiceSetServiceStatus[] | undefined {
    return this._services;
  }
}

export class ServiceSetServiceSpec {
  private _name: string;
  private _namespace: string;
  private _template: string;
  private _values: string;

  constructor(data: ServiceSetServiceSpecData) {
    this._name = data.name;
    this._namespace = data.namespace;
    this._template = data.template;
    this._values = data.values;
  }

  public get name(): string {
    return this._name;
  }

  public get namespace(): string {
    return this._namespace;
  }

  public get template(): string {
    return this._template;
  }

  public get values(): string {
    return this._values;
  }
}

export class ServiceSetServiceStatus implements Condition {
  private _type: string;
  private _lastTransitionTimeDate: Date;
  private _name: string;
  private _namespace: string;
  private _template: string;
  private _version: string;
  private _state: string;
  private _failureMessage?: string;

  constructor(data: ServiceSetServiceStatusData) {
    this._type = data.type;
    this._lastTransitionTimeDate = new Date(data.lastStateTransitionTime);
    this._name = data.name;
    this._namespace = data.namespace;
    this._template = data.template;
    this._version = data.version;
    this._state = data.state;
  }

  public get type(): string {
    return this._type;
  }

  public get modificationDate(): Date {
    return this._lastTransitionTimeDate;
  }

  public get name(): string {
    return this._name;
  }

  public get namespace(): string {
    return this._namespace;
  }

  public get template(): string {
    return this._template;
  }

  public get version(): string {
    return this._version;
  }

  public get status(): string {
    return this._state;
  }

  public get isHealthy(): boolean {
    return this._state === "Deployed";
  }

  public get reason(): string | undefined {
    return undefined;
  }

  public get message(): string | undefined {
    return this._failureMessage;
  }
}

export type ServiceSetData = K8sObjectData<ServiceSetSpecData, ServiceSetStatusData>;

export interface ServiceSetSpecData {
  cluster: string;
  multiClusterService?: string;
  services?: ServiceSetServiceSpecData[];
}

export interface ServiceSetServiceSpecData {
  name: string;
  namespace: string;
  template: string;
  values: string;
}

export interface ServiceSetStatusData {
  conditions: ClusterConditionData[];
  deployed: boolean;
  provider: {
    ready: boolean;
  };
  services?: ServiceSetServiceStatusData[];
}

export interface ServiceSetServiceStatusData {
  type: string;
  lastStateTransitionTime: string;
  name: string;
  namespace: string;
  template: string;
  version: string;
  state: string;
  failureMessage?: string;
}
