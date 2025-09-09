import { useClusterDeploymentsProvider } from "@/providers/ClusterDeploymentsProvider";
import { capitalizeFirstLetter } from "@/utils/formatter";
import { ClusterDeployment, ClusterDeploymentSet } from "./models";
import { DashboardData } from "@/components/pages/dashboards/DashboardTypes";
import { Server } from "lucide-react";
import {
  TableColAge,
  TableColName,
  TableColNamespace,
  TableColStatus
} from "../dashboards/TableColumns";
import {
  DetailMetadataTab,
  DetailRawJsonTab,
  DetailsNewTab,
  DetailStatusTab
} from "../dashboards/DetailTabs";
import DashboardList from "../dashboards/DashboardList";
import DashboardDetails from "../dashboards/DashboardDetails";
import ClusterDeploymentConfigurationTab from "./ClusterDeploymentConfigTab";

export const ClusterDeploymentDashboard = (): DashboardData<
  ClusterDeploymentSet,
  ClusterDeployment
> => {
  const dashboardData = {
    name: "Cluster Deployments",
    id: "cluster-deployments",
    store: useClusterDeploymentsProvider,
    icon: Server,
    tableCols: [
      TableColNamespace(),
      TableColName(),
      TableColStatus(),
      {
        head: { text: "Role", width: 100 },
        valueFn: (item: ClusterDeployment) => (
          <>{capitalizeFirstLetter(item.role ?? "N/A")}</>
        )
      },
      {
        head: { text: "Template", width: 180 },
        valueFn: (item: ClusterDeployment) => <>{item.spec.template}</>
      },
      TableColAge()
    ],
    detailTabs: [
      DetailStatusTab(),
      DetailsNewTab("Configuration", (item: ClusterDeployment) => (
        <ClusterDeploymentConfigurationTab clusterDeployment={item} />
      )),
      DetailMetadataTab(),
      DetailRawJsonTab()
    ]
  };

  return {
    ...dashboardData,
    renderDetails: () => <DashboardDetails {...dashboardData} />,
    renderList: () => <DashboardList {...dashboardData} />
  };
};
