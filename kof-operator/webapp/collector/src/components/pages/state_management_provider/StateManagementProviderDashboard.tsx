import { useStateManagementProvidersProvider } from "@/providers/StateManagementProvidersProvider";
import { StateManagementProvider, StateManagementProviderSet } from "./models";
import { DashboardData } from "@/components/pages/dashboards/DashboardTypes";
import { ServerCog } from "lucide-react";
import DashboardDetails from "../dashboards/DashboardDetails";
import DashboardList from "../dashboards/DashboardList";
import { TableColAge, TableColName, TableColStatus } from "../dashboards/TableColumns";
import {
  DetailMetadataTab,
  DetailRawJsonTab,
  DetailStatusTab
} from "../dashboards/DetailTabs";

export const StateManagementProviderDashboard = (): DashboardData<
  StateManagementProviderSet,
  StateManagementProvider
> => {
  const dashboardData = {
    name: "State Management Provider",
    id: "state-management-provider",
    store: useStateManagementProvidersProvider,
    icon: ServerCog,
    tableCols: [TableColName(), TableColStatus(), TableColAge()],
    detailTabs: [DetailStatusTab(), DetailMetadataTab(), DetailRawJsonTab()]
  };

  return {
    ...dashboardData,
    renderDetails: () => <DashboardDetails {...dashboardData} />,
    renderList: () => <DashboardList {...dashboardData} />
  };
};
