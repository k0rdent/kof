import { DefaultStatus } from "@/models/DefaultCondition";
import { K8sObjectData, K8sObject } from "@/models/k8sObject";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { Condition, Metadata } from "@/models/ObjectMeta";

export class ClusterSummariesSet extends K8sObjectSet<ClusterSummary> {
  protected createK8sObject(
    path: string,
    data: K8sObjectData<ClusterSummarySpecData, ClusterSummaryStatusData>
  ): ClusterSummary {
    return new ClusterSummary(path, data);
  }
}

export class ClusterSummary extends K8sObject<
  ClusterSummarySpec,
  ClusterSummaryStatus,
  ClusterSummarySpecData,
  ClusterSummaryStatusData
> {
  public get isHealthy(): boolean {
    const fs = this.status.featureSummaries.arr;
    const allFeaturesOk = fs.length === 0 || fs.every((f) => f.isHealthy);

    const hrs = this.status.helmReleaseSummaries.arr ?? [];
    const allHelmOk = hrs.length === 0 || hrs.every((r) => r.isHealthy);

    return allFeaturesOk && allHelmOk;
  }

  protected createSpec(data: ClusterSummarySpecData): ClusterSummarySpec {
    return new ClusterSummarySpec(data);
  }

  protected createStatus(data: ClusterSummaryStatusData): ClusterSummaryStatus {
    return new ClusterSummaryStatus(data);
  }
}

export class ClusterSummarySpec {
  private _clusterNamespace: string;
  private _clusterName: string;
  private _clusterType: string;
  private _clusterProfileSpec: ClusterProfileSpec;

  constructor(data: ClusterSummarySpecData) {
    this._clusterName = data.clusterName;
    this._clusterNamespace = data.clusterNamespace;
    this._clusterType = data.clusterType;
    this._clusterProfileSpec = data.clusterProfileSpec;
  }

  public get clusterName(): string {
    return this._clusterName;
  }

  public get clusterNamespace(): string {
    return this._clusterNamespace;
  }

  public get clusterType(): string {
    return this._clusterType;
  }

  public get clusterProfileSpec(): ClusterProfileSpec {
    return this._clusterProfileSpec;
  }
}

export class ClusterSummaryStatus implements DefaultStatus {
  private _featureSummaries: FeatureSummaries;
  private _helmReleaseSummaries: HelmReleaseSummaries;
  private _failureMessage: string | undefined;

  constructor(data: ClusterSummaryStatusData) {
    this._failureMessage = data.failureMessage;
    this._featureSummaries = new FeatureSummaries(data.featureSummaries);
    this._helmReleaseSummaries = new HelmReleaseSummaries(data.helmReleaseSummaries);
  }

  public get featureSummaries(): FeatureSummaries {
    return this._featureSummaries;
  }

  public get helmReleaseSummaries(): HelmReleaseSummaries {
    return this._helmReleaseSummaries;
  }

  public get conditions(): Condition[] {
    return this._featureSummaries.arr;
  }

  public get failureMessage(): string | string[] | undefined {
    if (this._failureMessage) return this._failureMessage;

    const unhealthyFeatureMessages = this._featureSummaries.arr
      .filter((fs) => !fs.isHealthy && fs.message)
      .map((fs) => fs.message ?? "");

    const unhealthyHelmMessages = this._helmReleaseSummaries.arr
      .filter((h) => !h.isHealthy && h.message)
      .map((h) => h.message ?? "");

    const allMessages: string[] = [
      ...unhealthyFeatureMessages,
      ...unhealthyHelmMessages
    ];

    if (allMessages.length > 0) return allMessages;
  }
}

export class FeatureSummaries {
  private _featureSummaries: FeatureSummary[] = [];
  private _healthyCount: number = 0;

  constructor(featureSummaries: FeatureSummaryData[]) {
    featureSummaries?.forEach((fs) =>
      this._featureSummaries.push(new FeatureSummary(fs))
    );

    this._healthyCount = this._featureSummaries.filter((fs) => fs.isHealthy).length;
  }

  public get count(): number {
    return this._featureSummaries.length;
  }

  public get healthyCount(): number {
    return this._healthyCount;
  }

  public get unhealthyCount(): number {
    return this.count - this._healthyCount;
  }

  public get arr(): FeatureSummary[] {
    return this._featureSummaries;
  }
}

export class FeatureSummary implements Condition {
  private _id: FeatureID;
  private _status: FeatureProvisionStatus;
  private _lastAppliedDate: Date;
  private _failureReason?: string;
  private _failureMessage?: string;
  private _hash: string;

  constructor(data: FeatureSummaryData) {
    this._failureReason = data.failureReason;
    this._failureMessage = data.failureMessage;
    this._status = data.status;
    this._id = data.featureID;
    this._hash = data.hash;
    this._lastAppliedDate = new Date(data.lastAppliedTime);
  }

  public get name(): string {
    return this._id;
  }

  public get status(): string {
    return this._status;
  }

  public get modificationDate(): Date {
    return this._lastAppliedDate;
  }

  public get isHealthy(): boolean {
    return this._status === "Provisioned";
  }

  public get reason(): string | undefined {
    return this._failureReason;
  }

  public get message(): string | undefined {
    return this._failureMessage;
  }

  public get hash(): string {
    return this._hash;
  }
}

export class HelmReleaseSummaries {
  private _helmReleaseSummaries: HelmReleaseSummary[] = [];
  private _healthyCount: number = 0;

  constructor(helmReleaseSummaries: HelmReleaseSummaryData[]) {
    helmReleaseSummaries?.forEach((hrs) =>
      this._helmReleaseSummaries.push(new HelmReleaseSummary(hrs))
    );

    this._healthyCount = this._helmReleaseSummaries.filter((fs) => fs.isHealthy).length;
  }

  public get count(): number {
    return this._helmReleaseSummaries.length;
  }

  public get healthyCount(): number {
    return this._healthyCount;
  }

  public get unhealthyCount(): number {
    return this.count - this._healthyCount;
  }

  public get arr(): HelmReleaseSummary[] {
    return this._helmReleaseSummaries;
  }
}

export class HelmReleaseSummary implements Condition {
  private _releaseName: string;
  private _releaseNamespace: string;
  private _status: HelmReleaseStatus;
  private _valuesHash: string;
  private _conflictMessage: string | undefined;

  constructor(data: HelmReleaseSummaryData) {
    this._releaseName = data.releaseName;
    this._releaseNamespace = data.releaseNamespace;
    this._status = data.status;
    this._valuesHash = data.valuesHash;
    this._conflictMessage = data.conflictMessage;
  }

  public get isHealthy(): boolean {
    return this._status === "Managing";
  }

  public get name(): string {
    return this._releaseName;
  }

  public get releaseNamespace(): string {
    return this._releaseNamespace;
  }

  public get status(): string {
    return this._status;
  }

  public get modificationDate(): Date | undefined {
    return undefined;
  }

  public get reason(): string | undefined {
    return undefined;
  }

  public get message(): string | undefined {
    return this._conflictMessage;
  }

  public get hash(): string {
    return this._valuesHash;
  }
}

export interface ClusterSummaryData {
  apiVersion?: string;
  kind?: string;
  metadata: Metadata;
  spec: ClusterSummarySpecData;
  status: ClusterSummaryStatusData;
}

interface K8sResourceRef {
  apiVersion: string;
  kind: string;
  namespace?: string;
  name: string;
}

interface TemplateResourceRef {
  identifier?: string;
  resource: K8sResourceRef;
}

interface PolicyRef {
  name: string;
  namespace?: string;
  kind?: string;
}

interface HelmChartSpec {
  repoURL: string;
  chartName: string;
  chartVersion?: string;
  releaseName: string;
  releaseNamespace: string;
  action?: "Install" | "Upgrade" | "Uninstall" | string;
  values?: unknown;
}

interface ClusterProfileSpec {
  syncMode?: string;
  continueOnConflict?: boolean;
  stopMatchingBehavior?: string;
  tier?: string;

  templateResourceRefs?: TemplateResourceRef[];
  policyRefs?: PolicyRef[];

  helmCharts?: HelmChartSpec[];
}

interface ClusterSummarySpecData {
  clusterNamespace: string;
  clusterName: string;
  clusterType: string;
  clusterProfileSpec: ClusterProfileSpec;
}

export type FeatureID = "Resources" | "Helm" | "Kustomize";

export type FeatureProvisionStatus =
  | "Provisioned"
  | "Provisioning"
  | "Failed"
  | "FailedNonRetriable"
  | "Removing"
  | "Removed";

interface FeatureSummaryData {
  featureID: FeatureID;
  status: FeatureProvisionStatus;
  lastAppliedTime: string;
  hash: string;
  failureReason?: string;
  failureMessage?: string;
}

export type HelmReleaseStatus = "Managing" | "Conflict";

interface HelmReleaseSummaryData {
  releaseName: string;
  releaseNamespace: string;
  status: HelmReleaseStatus;
  valuesHash: string;
  conflictMessage?: string;
}

interface ClusterSummaryStatusData {
  featureSummaries: FeatureSummaryData[];
  helmReleaseSummaries: HelmReleaseSummaryData[];
  failureMessage?: string;
}
