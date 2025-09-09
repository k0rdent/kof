import { K8sObject, K8sObjectData } from "./k8sObject";

export abstract class K8sObjectSet<K extends K8sObject> {
  private _map: Record<string, K> = {};
  private _array: K[] = [];

  private _length = 0;
  private _healthyCount = 0;
  private _unhealthyCount = 0;

  constructor(objects: Record<string, K8sObjectData<unknown, unknown>>) {
    Object.entries(objects).forEach(([key, data]) => {
      const obj = this.createK8sObject(data);
      this._map[key] = obj;
      this._array.push(obj);
    });

    this._length = this._array.length;
    this._healthyCount = this._array.filter((o) => o.isHealthy).length;
    this._unhealthyCount = this._length - this._healthyCount;
  }

  public get objectsMap(): Record<string, K> {
    return this._map;
  }

  public get objects(): ReadonlyArray<K> {
    return this._array;
  }

  public get length(): number {
    return this._length;
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

  public getObject(name: string): K | null {
    return this._map[name] ?? null;
  }

  protected abstract createK8sObject(data: K8sObjectData<unknown, unknown>): K;
}
