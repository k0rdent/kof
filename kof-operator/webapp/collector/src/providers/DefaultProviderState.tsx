
import { CollectorMetricsRecordsManager } from "./collectors_metrics/CollectorsMetricsRecordManager";
import { Cluster, CollectorMetricsSet, Pod } from "@/components/pages/collectorPage/models";

export interface DefaultProviderState {
  error?: Error;
  isLoading: boolean;
  data: CollectorMetricsSet | null;
  selectedCluster: Cluster | null;
  selectedPod: Pod | null;
  metricsHistory: CollectorMetricsRecordsManager;
  fetch: () => void;
  setSelectedCluster: (name: string) => void;
  setSelectedPod: (name: string) => void;
}
