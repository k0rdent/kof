import { DashboardData } from "../dashboards/DashboardTypes";
import { SveltosCluster, SveltosClusterSet } from "./models";
import {
  TableColAge,
  TableColName,
  TableColNamespace,
  TableColStatus,
} from "../dashboards/TableColumns";
import {
  DetailMetadataTab,
  DetailRawJsonTab,
  DetailStatusTab,
} from "../dashboards/DetailTabs";
import { Network } from "lucide-react";
import DashboardDetails from "../dashboards/DashboardDetails";
import DashboardList from "../dashboards/DashboardList";
import { useSveltosClusterProvider } from "@/providers/SveltosClusterProvider";

export const SveltosClusterDashboard = (): DashboardData<
  SveltosClusterSet,
  SveltosCluster
> => {
  const dashboardData = {
    name: "Sveltos Clusters",
    id: "sveltos-clusters",
    store: useSveltosClusterProvider,
    icon: Network,
    tableCols: [TableColNamespace(), TableColName(), TableColStatus(), TableColAge()],
    detailTabs: [DetailStatusTab(), DetailMetadataTab(), DetailRawJsonTab()],
  };

  return {
    ...dashboardData,
    renderDetails: () => <DashboardDetails {...dashboardData} />,
    renderList: () => <DashboardList {...dashboardData} />,
  };
};
