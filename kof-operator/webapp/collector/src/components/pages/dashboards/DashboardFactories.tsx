import { ClusterDeploymentDashboard } from "../cluster_deployments_page/ClusterDeploymentDashboard";
import { ClusterSummariesDashboard } from "../cluster_summaries_page/ClusterSummariesDashboard";
import { MultiClusterServiceDashboard } from "../multi_cluster_services_page/MultiClusterServiceDashboard";
import { ServiceSetsDashboard } from "../service_sets_page/ServiceSetsDashboard";
import { StateManagementProviderDashboard } from "../state_management_provider/StateManagementProviderDashboard";
import { SveltosClusterDashboard } from "../sveltos_cluster_page/SveltosClusterDashboard";

export const DashboardFactories = {
  SveltosClusterDashboard,
  ClusterDeploymentDashboard,
  ClusterSummariesDashboard,
  MultiClusterServiceDashboard,
  StateManagementProviderDashboard,
  ServiceSetsDashboard,
} as const;

type AllDashboardFactories = typeof DashboardFactories;
export type AllDashboards = ReturnType<
  AllDashboardFactories[keyof AllDashboardFactories]
>;

export const Dashboards: AllDashboards[] = Object.values(DashboardFactories).map(
  (factory) => factory(),
);
