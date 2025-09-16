import { Provider } from "./ProviderAbstract";
import {
  SveltosCluster,
  SveltosClusterData,
  SveltosClusterSet,
} from "@/components/pages/sveltos_cluster_page/models";

class SveltosClusterProvider extends Provider<
  SveltosClusterSet,
  SveltosCluster,
  SveltosClusterData
> {
  protected buildItems(
    rawItems: Record<string, SveltosClusterData>,
  ): SveltosClusterSet {
    return new SveltosClusterSet(rawItems);
  }

  protected selectFrom(items: SveltosClusterSet, name: string): SveltosCluster | null {
    return items.getObject(name);
  }
}

const sveltosClusterProvider = new SveltosClusterProvider(
  import.meta.env.VITE_SVELTOS_CLUSTERS_URL,
);
export const useSveltosClusterProvider = sveltosClusterProvider.use;
