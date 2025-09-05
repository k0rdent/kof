import {
  ClusterSummariesSet,
  ClusterSummary,
  ClusterSummaryData,
} from "@/components/pages/cluster_summaries_page/models";
import { Provider } from "./ProviderAbstract";

class ClusterSummariesProvider extends Provider<
  ClusterSummariesSet,
  ClusterSummary,
  ClusterSummaryData
> {
  protected buildItems(
    rawItems: Record<string, ClusterSummaryData>
  ): ClusterSummariesSet {
    return new ClusterSummariesSet(rawItems);
  }

  protected selectFrom(
    items: ClusterSummariesSet,
    name: string
  ): ClusterSummary | null {
    return items.getObject(name);
  }
}

const clusterSummariesProvider = new ClusterSummariesProvider(
  import.meta.env.VITE_CLUSTER_SUMMARIES_URL
);
export const useClusterSummariesProvider = clusterSummariesProvider.use;
