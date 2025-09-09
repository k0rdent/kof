import { useMultiClusterServiceProvider } from "@/providers/MultiClusterServicesProvider";
import { MultiClusterService, MultiClusterServiceSet } from "./models";
import { DashboardData } from "@/components/pages/dashboards/DashboardTypes";
import { Workflow } from "lucide-react";
import DashboardDetails from "../dashboards/DashboardDetails";
import DashboardList from "../dashboards/DashboardList";
import { TableColAge, TableColName, TableColStatus } from "../dashboards/TableColumns";
import {
  DetailMetadataTab,
  DetailRawJsonTab,
  DetailStatusTab
} from "../dashboards/DetailTabs";

export const MultiClusterServiceDashboard = (): DashboardData<
  MultiClusterServiceSet,
  MultiClusterService
> => {
  const dashboardData = {
    name: "Multi Cluster Service",
    id: "multi-cluster-service",
    store: useMultiClusterServiceProvider,
    icon: Workflow,
    tableCols: [
      TableColName(),
      TableColStatus(),
      {
        head: { text: "Services Ready", width: 150 },
        valueFn: (item: MultiClusterService) => (
          <>{item.getCondition("ServicesInReadyState")?.message}</>
        )
      },
      {
        head: { text: "Cluster Ready", width: 150 },
        valueFn: (item: MultiClusterService) => (
          <>{item.getCondition("ClusterInReadyState")?.message}</>
        )
      },
      TableColAge()
    ],
    detailTabs: [DetailStatusTab(), DetailMetadataTab(), DetailRawJsonTab()]
  };

  return {
    ...dashboardData,
    renderDetails: () => <DashboardDetails {...dashboardData} />,
    renderList: () => <DashboardList {...dashboardData} />
  };
};
