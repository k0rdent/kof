import { ClusterConditionData, Condition } from "./ObjectMeta";

export class DefaultCondition implements Condition {
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
