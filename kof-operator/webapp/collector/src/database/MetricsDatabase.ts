import { PodsMap } from "@/components/pages/collectorPage/models";
import Dexie, { type EntityTable } from "dexie";

type RawMetricsRecord = {
  timestamp: number;
  record: Record<string, PodsMap>;
};

interface Db extends Dexie {
  metrics: EntityTable<RawMetricsRecord, "timestamp">;
}

export class MetricsDatabase {
  private _db: Db;

  constructor() {
    this._db = new Dexie("MetricsDatabase") as Db;

    this._db.version(1).stores({
      metrics: "++timestamp, record",
    });
  }

  public async getRecords(
    minTime: number,
    maxTime: number
  ): Promise<RawMetricsRecord[]> {
    try {
      return await this._db.metrics
        .where("timestamp")
        .between(minTime, maxTime)
        .toArray();
    } catch (err) {
      console.error("Failed to get records from DB", err);
      return [];
    }
  }

  public async addRecord(
    timestamp: number,
    record: Record<string, PodsMap>
  ): Promise<void> {
    try {
      await this._db.metrics.add({
        timestamp,
        record,
      });
    } catch (err) {
      console.error("Failed to add record to DB", err);
    }
  }

  public async deleteOldRecords(timeUnix: number): Promise<void> {
    try {
      await this._db.metrics.where("timestamp").below(timeUnix).delete();
    } catch (err) {
      console.error("Failed to delete old records", err);
    }
  }

  public async deleteRecord(timestamp: number): Promise<void> {
    try {
      await this._db.metrics.delete(timestamp);
    } catch (err) {
      console.error("Failed to delete record", err);
    }
  }
}
