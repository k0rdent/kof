export class ObjectMeta {
  private _name: string;
  private _namespace: string;
  private _generation: number;
  private _creationDate: Date;
  private _labels: Record<string, string>;
  private _annotations: Record<string, string>;
  private _ownerReferences: OwnerReference[] | undefined;
  private _deletionDate: Date | undefined;

  constructor(data: Metadata) {
    this._name = data.name;
    this._namespace = data.namespace;
    this._generation = data.generation;
    this._labels = data.labels;
    this._annotations = data.annotations;
    this._ownerReferences = data.ownerReferences;
    this._creationDate = new Date(data.creationTimestamp);
    this._deletionDate = data.deletionTimestamp
      ? new Date(data.deletionTimestamp)
      : undefined;
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

  public get creationDate(): Date {
    return this._creationDate;
  }

  public get deletionDate(): Date | undefined {
    return this._deletionDate;
  }

  public get labels(): Record<string, string> {
    return this._labels;
  }

  public get annotations(): Record<string, string> {
    return this._annotations;
  }

  public get ownerReferences(): OwnerReference[] | undefined {
    return this._ownerReferences;
  }
}

export interface Metadata {
  name: string;
  namespace: string;
  generation: number;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  creationTimestamp: Date;
  deletionTimestamp?: Date;
  ownerReferences?: OwnerReference[];
}

export interface OwnerReference {
  apiVersion: string;
  kind: string;
  name: string;
}

export interface Condition {
  name: string;
  status: string;
  isHealthy: boolean;
  modificationDate?: Date;
  reason?: string;
  message?: string;
}

export interface ClusterConditionData {
  type: string;
  status: string;
  observedGeneration?: number;
  lastTransitionTime: string;
  reason: string;
  message: string;
}
