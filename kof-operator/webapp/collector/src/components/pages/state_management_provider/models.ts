import { ClusterConditionData, Metadata } from "@/models/ObjectMeta";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { K8sObject, K8sObjectData } from "@/models/k8sObject";
import { DefaultCondition, DefaultStatus } from "@/models/DefaultCondition";

export class StateManagementProviderSet extends K8sObjectSet<StateManagementProvider> {
  protected createK8sObject(
    path: string,
    data: K8sObjectData<
      StateManagementProviderSpecData,
      StateManagementProviderStatusData
    >,
  ): StateManagementProvider {
    return new StateManagementProvider(path, data);
  }
}

export class StateManagementProvider extends K8sObject<
  StateManagementProviderSpec,
  StateManagementProviderStatus,
  StateManagementProviderSpecData,
  StateManagementProviderStatusData
> {
  public get isHealthy(): boolean {
    return this.status.conditions.find((c) => !c.isHealthy) === undefined;
  }

  protected createSpec(
    data: StateManagementProviderSpecData,
  ): StateManagementProviderSpec {
    return new StateManagementProviderSpec(data);
  }

  protected createStatus(
    data: StateManagementProviderStatusData,
  ): StateManagementProviderStatus {
    return new StateManagementProviderStatus(data);
  }
}

export class StateManagementProviderSpec {
  private _selector: LabelSelector | undefined;
  private _adapter: ResourceReference;
  private _provisioner: ResourceReference[];
  private _provisionerCRDs: ProvisionerCRD[];
  private _suspend: boolean;

  constructor(data: StateManagementProviderSpecData) {
    this._adapter = data.adapter;
    this._provisioner = data.provisioner;
    this._selector = data.selector;
    this._provisionerCRDs = data.provisionerCRDs;
    this._suspend = data.suspend;
  }

  public get adapter(): ResourceReference {
    return this._adapter;
  }

  public get selector(): LabelSelector | undefined {
    return this._selector;
  }

  public get provisioner(): ResourceReference[] {
    return this._provisioner;
  }

  public get provisionerCRDs(): ProvisionerCRD[] {
    return this._provisionerCRDs;
  }

  public get suspend(): boolean {
    return this._suspend;
  }
}

export class StateManagementProviderStatus implements DefaultStatus {
  private _conditions: DefaultCondition[];
  private _ready: boolean;

  private _conditionsHealthyCount: number;
  private _conditionsUnhealthyCount: number;

  constructor(data: StateManagementProviderStatusData) {
    this._conditions = (data.conditions || []).map((c) => new DefaultCondition(c));
    this._ready = data.ready;

    this._conditionsHealthyCount = this._conditions.filter((c) => c.isHealthy).length;
    this._conditionsUnhealthyCount =
      this._conditions.length - this._conditionsHealthyCount;
  }

  public get conditions(): DefaultCondition[] {
    return this._conditions;
  }

  public get ready(): boolean {
    return this._ready;
  }

  public get conditionsHealthyCount(): number {
    return this._conditionsHealthyCount;
  }

  public get conditionsUnhealthyCount(): number {
    return this._conditionsUnhealthyCount;
  }
}

export interface StateManagementProviderData {
  apiVersion?: string;
  kind?: string;
  metadata: Metadata;
  spec: StateManagementProviderSpecData;
  status: StateManagementProviderStatusData;
}

export interface StateManagementProviderStatusData {
  conditions: ClusterConditionData[];
  ready: boolean;
}

export interface StateManagementProviderSpecData {
  selector?: LabelSelector;
  adapter: ResourceReference;
  provisioner: ResourceReference[];
  provisionerCRDs: ProvisionerCRD[];
  suspend: boolean;
}

interface LabelSelector {
  matchLabels: Record<string, string>;
  matchExpressions: LabelSelectorRequirement;
}

interface LabelSelectorRequirement {
  key: string;
  values: string[];
  operator: string;
}

interface ResourceReference {
  apiVersion: string;
  kind: string;
  name: string;
  namespace: string;
  readinessRule: string;
}

interface ProvisionerCRD {
  group: string;
  version: string;
  resources: string[];
}
