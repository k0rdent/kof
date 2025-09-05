import { ClusterConditionData, Condition, Metadata } from "@/models/ObjectMeta";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { K8sObject, K8sObjectData } from "@/models/k8sObject";

type MultiClusterServicesConditionType =
  | "ServicesInReadyState"
  | "ClusterInReadyState";

export class MultiClusterServiceSet extends K8sObjectSet<MultiClusterService> {
  protected createK8sObject(
    data: K8sObjectData<
      MultiClusterServiceSpecData,
      MultiClusterServiceStatusData
    >
  ): MultiClusterService {
    return new MultiClusterService(data);
  }
}

export class MultiClusterService extends K8sObject<
  MultiClusterServiceSpec,
  MultiClusterServiceStatus,
  MultiClusterServiceSpecData,
  MultiClusterServiceStatusData
> {
  public get isHealthy(): boolean {
    return this.status.conditions.find((c) => !c.isHealthy) === undefined;
  }

  public getCondition(type: MultiClusterServicesConditionType) {
    return this.status.getConditionByType(type);
  }
  protected createSpec(
    data: MultiClusterServiceSpecData
  ): MultiClusterServiceSpec {
    return new MultiClusterServiceSpec(data);
  }

  protected createStatus(
    data: MultiClusterServiceStatusData
  ): MultiClusterServiceStatus {
    return new MultiClusterServiceStatus(data);
  }
}

class MultiClusterServiceSpec {
  clusterSelector?: ClusterSelectorData;
  serviceSpec?: ServiceSpecData;

  constructor(data: MultiClusterServiceSpecData) {
    this.clusterSelector = data.clusterSelector;
    this.serviceSpec = data.serviceSpec
  }
}

class MultiClusterServiceStatus {
  public services: ServiceStatusData[] = [];
  public conditions: Condition[] = [];
  public observedGeneration?: number;

  private _conditionsHealthyCount: number;
  private _conditionsUnhealthyCount: number;

  constructor(data: MultiClusterServiceStatusData) {
    // this.services = (data.services || []).map((s) => new ServiceStatus(s));
    this.services = data.services ?? [];
    this.conditions = (data.conditions || []).map(
      (c) => new MultiClusterServiceCondition(c)
    );
    this.observedGeneration = data.observedGeneration;

    this._conditionsHealthyCount = this.conditions.filter(
      (c) => c.isHealthy
    ).length;
    this._conditionsUnhealthyCount =
      this.conditions.length - this._conditionsHealthyCount;
  }

  public getConditionByType(
    type: MultiClusterServicesConditionType
  ): Condition | undefined {
    return this.conditions.find((c) => c.name === type);
  }

  public get conditionsHealthyCount(): number {
    return this._conditionsHealthyCount;
  }

  public get conditionsUnhealthyCount(): number {
    return this._conditionsUnhealthyCount;
  }
}

class MultiClusterServiceCondition implements Condition {
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

  public get name(): string {
    return this._type;
  }

  public get status(): string {
    return this._status == "True" ? "Ready" : "Failed";
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

  public get modificationDate(): Date {
    return this._lastTransitionTimeDate;
  }

  public get isHealthy(): boolean {
    return this._status === "True";
  }
}

export interface MultiClusterServiceData {
  apiVersion?: string;
  kind?: string;
  metadata: Metadata;
  spec: MultiClusterServiceSpecData;
  status: MultiClusterServiceStatusData;
}

interface MultiClusterServiceSpecData {
  clusterSelector?: ClusterSelectorData;
  serviceSpec?: ServiceSpecData;
}

interface ClusterSelectorData {
  matchLabels?: Record<string, string>;
  matchExpressions?: MatchExpressionData[];
}

interface MatchExpressionData {
  key: string;
  operator: string;
  values?: string[];
}

interface ServiceSpecData {
  syncMode?: string;
  provider?: Record<string, unknown>;
  services?: ServiceSpecItemData[];
  templateResourceRefs?: TemplateResourceRefData[];
  priority?: number;
}

interface ServiceSpecItemData {
  values?: string;
  template: string;
  templateChain?: string;
  name: string;
  namespace: string;
  valuesFrom?: ValuesFromData[];
}

interface ValuesFromData {
  kind: string;
  name: string;
}

interface TemplateResourceRefData {
  resource: ResourceRefData;
  identifier: string;
  optional?: boolean;
}

interface ResourceRefData {
  kind: string;
  namespace: string;
  name: string;
  apiVersion: string;
}

interface MultiClusterServiceStatusData {
  services?: ServiceStatusData[];
  conditions?: ClusterConditionData[];
  observedGeneration?: number;
}

interface ServiceStatusData {
  type: string;
  name: string;
  namespace: string;
  template: string;
  version: string;
  state: string;
  failureMessage?: string;
  lastStateTransitionTime?: string;
  conditions?: ClusterConditionData[];
}
