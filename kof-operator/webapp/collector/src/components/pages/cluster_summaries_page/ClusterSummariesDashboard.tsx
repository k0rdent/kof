import { useClusterSummariesProvider } from "@/providers/ClusterSummariesProvider";
import { ClusterSummariesSet, ClusterSummary } from "./models";
import { Button } from "@/components/generated/ui/button";
import { DashboardData } from "@/components/pages/dashboards/DashboardTypes";
import { Layers, MoveRight } from "lucide-react";
import { Link } from "react-router-dom";
import StatusTab from "@/components/shared/tabs/StatusTab";
import DashboardDetails from "../dashboards/DashboardDetails";
import DashboardList from "../dashboards/DashboardList";
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

export const ClusterSummariesDashboard = (): DashboardData<
  ClusterSummariesSet,
  ClusterSummary
> => {
  const dashboardData: DashboardData<ClusterSummariesSet, ClusterSummary> = {
    name: "Cluster Summaries",
    id: "cluster-summaries",
    store: useClusterSummariesProvider,
    icon: Layers,
    tableCols: [TableColNamespace(), TableColName(), TableColStatus(), TableColAge()],
    detailsHeaderChild: (item: ClusterSummary) => (
      <Link
        to={`/cluster-deployments/${item.spec.clusterNamespace}/${item.spec.clusterName}`}
      >
        <Button variant="outline" className="cursor-pointer">
          <span>Go to Cluster Deployment</span>
          <MoveRight />
        </Button>
      </Link>
    ),
    detailTabs: [
      DetailStatusTab(),
      DetailsNewTab(
        "Helm Charts",
        (item: ClusterSummary) => (
          <StatusTab conditions={item.status.helmReleaseSummaries.arr} />
        ),
        (item: ClusterSummary) => item.status.helmReleaseSummaries.arr.length === 0
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
