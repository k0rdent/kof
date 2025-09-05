import AppSidebar from "./components/features/AppSidebar";
import MainPage from "./components/features/MainPage";
import CollectorMetricsPage from "./components/pages/collectorPage/CollectorPage";
import NoPage from "./components/pages/NoPage";
import {
  SidebarProvider,
  SidebarTrigger,
} from "@/components/generated/ui/sidebar";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { useCollectorMetricsState } from "./providers/collectors_metrics/CollectorsMetricsProvider";
import { useEffect } from "react";
import { useVictoriaMetricsState } from "./providers/victoria_metrics/VictoriaMetricsProvider";
import CollectorContent from "./components/pages/collectorPage/components/collector-details/CollectorContent";
import VictoriaPage from "./components/pages/victoriaPage/VictoriaPage";
import VictoriaDetailsPage from "./components/pages/victoriaPage/victoria-details/VictoriaDetailsPage";
import ClusterDeploymentLayout from "./components/pages/clusterDeploymentsPage/ClusterDeploymentLayout";
import ClusterDeploymentDetails from "./components/pages/clusterDeploymentsPage/components/details/ClusterDeploymentDetails";
import ClusterDeploymentsList from "./components/pages/clusterDeploymentsPage/components/list/ClusterDeploymentList";
import ClusterSummaryLayout from "./components/pages/cluster_summaries_page/ClusterSummariesLayout";
import ClusterSummariesList from "./components/pages/cluster_summaries_page/list/ClusterSummariesList";
import ClusterSummaryDetails from "./components/pages/cluster_summaries_page/details/ClusterSummaryDetails";
import MultiClusterServicesLayout from "./components/pages/multi_cluster_services_page/MultiClusterServicesLayout";
import MultiClusterServicesList from "./components/pages/multi_cluster_services_page/list/MultiClusterServicesList";
import MultiClusterServiceDetails from "./components/pages/multi_cluster_services_page/details/MultiClusterServiceDetails";
import StateManagementProviderLayout from "./components/pages/state_management_provider/StateManagementProviderLayout";
import StateManagementProviderList from "./components/pages/state_management_provider/StateManagementProviderList";
import StateManagementProviderDetails from "./components/pages/state_management_provider/StateManagementProviderDetails";

function App() {
  const { fetch: fetchCollector, isLoading: collectorIsLoading } =
    useCollectorMetricsState();
  useEffect(() => {
    const intervalId = setInterval(() => {
      if (!collectorIsLoading) {
        fetchCollector();
      }
    }, 20 * 1000);
    return () => clearInterval(intervalId);
  }, [fetchCollector, collectorIsLoading]);

  const { fetch, isLoading } = useVictoriaMetricsState();
  useEffect(() => {
    const intervalId = setInterval(() => {
      if (!isLoading) {
        fetch();
      }
    }, 20 * 1000);
    return () => clearInterval(intervalId);
  }, [fetch, isLoading]);

  return (
    <BrowserRouter>
      <SidebarProvider>
        <AppSidebar />
        <main className="flex flex-col w-full min-h-screen">
          <SidebarTrigger />
          <Routes>
            <Route path="/" element={<MainPage />} />
            <Route path="collectors" element={<CollectorMetricsPage />} />
            <Route
              path="collectors/:cluster/:collector"
              element={<CollectorContent />}
            />

            <Route path="victoria" element={<VictoriaPage />} />
            <Route
              path="victoria/:cluster/:pod"
              element={<VictoriaDetailsPage />}
            />

            <Route
              path="cluster-deployments"
              element={<ClusterDeploymentLayout />}
            >
              <Route index element={<ClusterDeploymentsList />} />
              <Route
                path=":clusterName"
                element={<ClusterDeploymentDetails />}
              />
            </Route>

            <Route path="cluster-summaries" element={<ClusterSummaryLayout />}>
              <Route index element={<ClusterSummariesList />} />
              <Route path=":summaryName" element={<ClusterSummaryDetails />} />
            </Route>

            <Route
              path="multi-cluster-services"
              element={<MultiClusterServicesLayout />}
            >
              <Route index element={<MultiClusterServicesList />} />
              <Route
                path=":serviceName"
                element={<MultiClusterServiceDetails />}
              />
            </Route>

            <Route
              path="state-management-providers"
              element={<StateManagementProviderLayout />}
            >
              <Route index element={<StateManagementProviderList />} />
              <Route
                path=":providerName"
                element={<StateManagementProviderDetails />}
              />
            </Route>

            <Route path="*" element={<NoPage />} />
          </Routes>
        </main>
      </SidebarProvider>
    </BrowserRouter>
  );
}

export default App;
