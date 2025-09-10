import { useServiceSetsProvider } from "@/providers/ServiceSetsProvider";
import { DashboardData } from "../dashboards/DashboardTypes";
import { ServiceSet, ServiceSetListSet } from "./models";
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
import { Boxes } from "lucide-react";
import DashboardDetails from "../dashboards/DashboardDetails";
import DashboardList from "../dashboards/DashboardList";
import StatusTab from "@/components/shared/tabs/StatusTab";

export const ServiceSetsDashboard = (): DashboardData<
  ServiceSetListSet,
  ServiceSet
> => {
  const dashboardData = {
    name: "Service Sets",
    id: "service-sets",
    store: useServiceSetsProvider,
    icon: Boxes,
    tableCols: [TableColNamespace(), TableColName(), TableColStatus(), TableColAge()],
    detailTabs: [
      DetailStatusTab(),
      DetailsNewTab(
        "Services",
        (i: ServiceSet) => <StatusTab conditions={i.status.services ?? []} />,
        (item: ServiceSet) => !item.status.services || item.status.services.length === 0
      ),
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
