import { Provider } from "./ProviderAbstract";
import {
  StateManagementProvider,
  StateManagementProviderData,
  StateManagementProviderSet,
} from "@/components/pages/state_management_provider/models";

class StateManagementProvidersProvider extends Provider<
  StateManagementProviderSet,
  StateManagementProvider,
  StateManagementProviderData
> {
  protected buildItems(
    rawItems: Record<string, StateManagementProviderData>
  ): StateManagementProviderSet {
    return new StateManagementProviderSet(rawItems);
  }
  protected selectFrom(
    items: StateManagementProviderSet,
    name: string
  ): StateManagementProvider | null {
    return items.getObject(name);
  }
}

const stateManagementProvidersProvider = new StateManagementProvidersProvider(
  import.meta.env.VITE_STATE_MANAGEMENT_PROVIDERS_URL
);
export const useStateManagementProvidersProvider = stateManagementProvidersProvider.use;
