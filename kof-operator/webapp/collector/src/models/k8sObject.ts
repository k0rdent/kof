import { DefaultStatus } from "./DefaultCondition";
import { Metadata, ObjectMeta } from "./ObjectMeta";

export abstract class K8sObject<
  Spec = unknown,
  Status extends DefaultStatus = DefaultStatus,
  SpecRaw = unknown,
  StatusRaw = unknown,
> {
  private _metadata: ObjectMeta;
  private _spec: Spec;
  private _status: Status;
  private _rawData: K8sObjectData<SpecRaw, StatusRaw>;
  private _clusterName: string;

  constructor(path: string, data: K8sObjectData<SpecRaw, StatusRaw>) {
    console.log("K8sObject constructor called with path:", path, "and data:", data);
    this._clusterName = path.split("/")[0];
    this._spec = this.createSpec(data.spec);
    this._status = this.createStatus(data.status);
    this._metadata = new ObjectMeta(data.metadata);
    this._rawData = data;
  }

  public get name(): string {
    return this._metadata.name;
  }

  public get namespace(): string {
    return this._metadata.namespace;
  }

  public get clusterName(): string {
    return this._clusterName;
  }

  public get raw(): K8sObjectData<SpecRaw, StatusRaw> {
    return this._rawData;
  }

  public get metadata(): ObjectMeta {
    return this._metadata;
  }

  public get spec(): Spec {
    return this._spec;
  }

  public get status(): Status {
    return this._status;
  }

  public get ageInSeconds(): number {
    return (Date.now() - this._metadata.creationDate.getTime()) / 1000;
  }

  public abstract get isHealthy(): boolean;

  protected abstract createSpec(raw: SpecRaw): Spec;
  protected abstract createStatus(raw: StatusRaw): Status;
}

export interface K8sObjectData<SpecRaw, StatusRaw> {
  apiVersion?: string;
  kind?: string;
  metadata: Metadata;
  spec: SpecRaw;
  status: StatusRaw;
}
