import { ClusterConditionData, Condition, Metadata } from "@/models/ObjectMeta";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { K8sObject, K8sObjectData } from "@/models/k8sObject";
import { DefaultCondition, DefaultStatus } from "@/models/DefaultCondition";

type MultiClusterServicesConditionType = "ServicesInReadyState" | "ClusterInReadyState";

export class MultiClusterServiceSet extends K8sObjectSet<MultiClusterService> {
  protected createK8sObject(
    path: string,
    data: K8sObjectData<MultiClusterServiceSpecData, MultiClusterServiceStatusData>
  ): MultiClusterService {
    return new MultiClusterService(path, data);
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
  protected createSpec(data: MultiClusterServiceSpecData): MultiClusterServiceSpec {
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
    this.serviceSpec = data.serviceSpec;
  }
}

class MultiClusterServiceStatus implements DefaultStatus {
  public services: ServiceStatusData[] = [];
  public conditions: Condition[] = [];
  public observedGeneration?: number;

  private _conditionsHealthyCount: number;
  private _conditionsUnhealthyCount: number;

  constructor(data: MultiClusterServiceStatusData) {
    this.services = data.services ?? [];
    this.conditions = (data.conditions || []).map((c) => new DefaultCondition(c));
    this.observedGeneration = data.observedGeneration;

    this._conditionsHealthyCount = this.conditions.filter((c) => c.isHealthy).length;
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
