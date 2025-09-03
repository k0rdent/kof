import { Condition, Metadata, ObjectMeta } from "@/models/ObjectMeta";

export class ClusterSummariesSet {
  private _summaries: Record<string, ClusterSummary> = {};
  private _summariesArr: ClusterSummary[] = [];
  private _healthyCount: number;
  private _unhealthyCount: number;

  constructor(data: ClusterSummariesSetData) {
    Object.entries(data.clusterSummaries).forEach(([k, v]) => {
      const cs: ClusterSummary = new ClusterSummary(v);
      this.clusterSummaries[k] = cs;
      this._summariesArr.push(cs);
    });

    this._healthyCount = this._summariesArr.filter((s) => s.isHealthy).length;
    this._unhealthyCount = this._summariesArr.length - this._healthyCount;
  }

  public get clusterSummaries(): Record<string, ClusterSummary> {
    return this._summaries;
  }

  public get clusterSummariesArray(): ClusterSummary[] {
    return this._summariesArr;
  }

  public get length(): number {
    return this._summariesArr.length;
  }

  public get unhealthyCount(): number {
    return this._unhealthyCount;
  }

  public get healthyCount(): number {
    return this._healthyCount;
  }

  public get isHealthy(): boolean {
    return this._unhealthyCount === 0;
  }

  public getSummary(name: string): ClusterSummary | undefined {
    return this._summaries[name];
  }
}

export class ClusterSummary {
  private _metadata: ObjectMeta;
  private _status: ClusterSummaryStatus;
  private _spec: ClusterSummarySpec;
  private _rawData: ClusterSummaryData;

  constructor(data: ClusterSummaryData) {
    this._status = new ClusterSummaryStatus(data.status);
    this._metadata = new ObjectMeta(data.metadata);
    this._spec = data.spec;
    this._rawData = data;
  }

  public get name(): string {
    return this._metadata.name;
  }

  public get namespace(): string {
    return this._metadata.namespace;
  }

  public get creationDate(): Date {
    return this._metadata.creationDate;
  }

  public get generation(): number {
    return this._metadata.generation;
  }

  public get labels(): Record<string, string> {
    return this._metadata.labels;
  }

  public get annotations(): Record<string, string> {
    return this._metadata.annotations;
  }

  public get status(): ClusterSummaryStatus {
    return this._status;
  }

  public get metadata(): ObjectMeta {
    return this._metadata;
  }

  public get spec(): ClusterSummarySpec {
    return this._spec;
  }

  public get rawData(): ClusterSummaryData {
    return this._rawData;
  }

  public get isHealthy(): boolean {
    const fs = this._status.featureSummaries.arr;
    const allFeaturesOk = fs.length === 0 || fs.every((f) => f.isHealthy);

    const hrs = this._status.helmReleaseSummaries.arr ?? [];
    const allHelmOk = hrs.length === 0 || hrs.every((r) => r.isHealthy);

    return allFeaturesOk && allHelmOk;
  }

  public get ageInSeconds(): number {
    const timeNow: number = Date.now();
    const creationTime: number = this.creationDate.getTime();
    return (timeNow - creationTime) / 1000;
  }
}

export class ClusterSummaryStatus {
  private _featureSummaries: FeatureSummaries;
  private _helmReleaseSummaries: HelmReleaseSummaries;
  private _failureMessage: string | undefined;

  constructor(data: ClusterSummaryStatusData) {
    this._failureMessage = data.failureMessage;
    this._featureSummaries = new FeatureSummaries(data.featureSummaries);
    this._helmReleaseSummaries = new HelmReleaseSummaries(
      data.helmReleaseSummaries
    );
  }

  public get featureSummaries(): FeatureSummaries {
    return this._featureSummaries;
  }

  public get helmReleaseSummaries(): HelmReleaseSummaries {
    return this._helmReleaseSummaries;
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
      ...unhealthyHelmMessages,
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

    this._healthyCount = this._featureSummaries.filter(
      (fs) => fs.isHealthy
    ).length;
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

    this._healthyCount = this._helmReleaseSummaries.filter(
      (fs) => fs.isHealthy
    ).length;
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
  spec: ClusterSummarySpec;
  status: ClusterSummaryStatusData;
}

export interface ClusterSummariesSetData {
  clusterSummaries: Record<string, ClusterSummaryData>;
}

export interface K8sResourceRef {
  apiVersion: string;
  kind: string;
  namespace?: string;
  name: string;
}

export interface TemplateResourceRef {
  identifier?: string;
  resource: K8sResourceRef;
}

export interface PolicyRef {
  name: string;
  namespace?: string;
  kind?: string;
}

export interface HelmChartSpec {
  repoURL: string;
  chartName: string;
  chartVersion?: string;
  releaseName: string;
  releaseNamespace: string;
  action?: "Install" | "Upgrade" | "Uninstall" | string;
  values?: unknown;
}

export interface ClusterProfileSpec {
  syncMode?: string;
  continueOnConflict?: boolean;
  stopMatchingBehavior?: string;
  tier?: string;

  templateResourceRefs?: TemplateResourceRef[];
  policyRefs?: PolicyRef[];

  helmCharts?: HelmChartSpec[];
}

export interface ClusterSummarySpec {
  clusterNamespace?: string;
  clusterName?: string;
  clusterType?: string;
  clusterProfileSpec?: ClusterProfileSpec;
}

export type FeatureID = "Resources" | "Helm" | "Kustomize";

export type FeatureProvisionStatus =
  | "Provisioned"
  | "Provisioning"
  | "Failed"
  | "FailedNonRetriable"
  | "Removing"
  | "Removed";

export interface FeatureSummaryData {
  featureID: FeatureID;
  status: FeatureProvisionStatus;
  lastAppliedTime: string;
  hash: string;
  failureReason?: string;
  failureMessage?: string;
}

export type HelmReleaseStatus = "Managing" | "Conflict";

export interface HelmReleaseSummaryData {
  releaseName: string;
  releaseNamespace: string;
  status: HelmReleaseStatus;
  valuesHash: string;
  conflictMessage?: string;
}

export interface ClusterSummaryStatusData {
  featureSummaries: FeatureSummaryData[];
  helmReleaseSummaries: HelmReleaseSummaryData[];
  failureMessage?: string;
}
