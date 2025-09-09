import {
  ServiceSet,
  ServiceSetData,
  ServiceSetListSet
} from "@/components/pages/service_sets_page/models";
import { Provider } from "./ProviderAbstract";

class ServiceSetsProvider extends Provider<
  ServiceSetListSet,
  ServiceSet,
  ServiceSetData
> {
  protected buildItems(rawItems: Record<string, ServiceSetData>): ServiceSetListSet {
    return new ServiceSetListSet(rawItems);
  }
  protected selectFrom(items: ServiceSetListSet, name: string): ServiceSet | null {
    return items.getObject(name);
  }
}

const serviceSetsProvider = new ServiceSetsProvider(
  import.meta.env.VITE_SERVICE_SETS_URL
);
export const useServiceSetsProvider = serviceSetsProvider.use;
