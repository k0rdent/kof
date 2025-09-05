import {
  MultiClusterService,
  MultiClusterServiceData,
  MultiClusterServiceSet,
} from "@/components/pages/multi_cluster_services_page/models";
import { Provider } from "./ProviderAbstract";

class MultiClusterServicesProvider extends Provider<
  MultiClusterServiceSet,
  MultiClusterService,
  MultiClusterServiceData
> {
  protected buildItems(
    rawItems: Record<string, MultiClusterServiceData>
  ): MultiClusterServiceSet {
    return new MultiClusterServiceSet(rawItems);
  }

  protected selectFrom(
    items: MultiClusterServiceSet,
    name: string
  ): MultiClusterService | null {
    return items.getObject(name);
  }
}

const multiClusterServiceProvider = new MultiClusterServicesProvider(
  import.meta.env.VITE_MULTI_CLUSTER_SERVICE_URL
);
export const useMultiClusterServiceProvider = multiClusterServiceProvider.use;
