import {
  ClusterDeployment,
  ClusterDeploymentData,
  ClusterDeploymentSet,
} from "@/components/pages/clusterDeploymentsPage/models";
import { Provider } from "./ProviderAbstract";

class ClusterSummariesProvider extends Provider<
  ClusterDeploymentSet,
  ClusterDeployment,
  ClusterDeploymentData
> {
  protected buildItems(
    rawItems: Record<string, ClusterDeploymentData>
  ): ClusterDeploymentSet {
    return new ClusterDeploymentSet(rawItems);
  }

  protected selectFrom(
    items: ClusterDeploymentSet,
    name: string
  ): ClusterDeployment | null {
    return items.getObject(name);
  }
}

const clusterDeploymentsProvider = new ClusterSummariesProvider(
  import.meta.env.VITE_CLUSTER_DEPLOYMENTS_URL
);
export const useClusterDeploymentsProvider = clusterDeploymentsProvider.use;
