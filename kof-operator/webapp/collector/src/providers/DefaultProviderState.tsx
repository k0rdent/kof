
import { MetricsRecordsManager } from "./collectors_metrics/CollectorsMetricsRecordManager";
import { Cluster, ClustersSet, Pod } from "@/components/pages/collectorPage/models";

export interface DefaultProviderState {
  error?: Error;
  isLoading: boolean;
  data: ClustersSet | null;
  selectedCluster: Cluster | null;
  selectedPod: Pod | null;
  metricsHistory: MetricsRecordsManager;
  fetch: () => Promise<void>;
  setSelectedCluster: (name: string) => void;
  setSelectedPod: (name: string) => void;
}
