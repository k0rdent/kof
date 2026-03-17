import AppSidebar from "./components/features/AppSidebar";
import MainPage from "./components/features/MainPage";
import CollectorMetricsPage from "./components/pages/collectorPage/CollectorPage";
import NoPage from "./components/pages/NoPage";
import { SidebarProvider, SidebarTrigger } from "@/components/generated/ui/sidebar";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { useCollectorMetricsState } from "./providers/collectors_metrics/CollectorsMetricsProvider";
import { useEffect } from "react";
import { useVictoriaMetricsState } from "./providers/victoria_metrics/VictoriaMetricsProvider";
import { Dashboards } from "./components/pages/dashboards/DashboardFactories";
import CollectorContent from "./components/pages/collectorPage/components/collector-details/CollectorContent";
import VictoriaPage from "./components/pages/victoriaPage/VictoriaPage";
import VictoriaDetailsPage from "./components/pages/victoriaPage/victoria-details/VictoriaDetailsPage";
import DashboardLayout from "./components/pages/dashboards/DashboardLayout";
import IstioSecretsPage from "./components/pages/istio/IstioSecretsPage";
import IstioMeshesPage from "./components/pages/istio/IstioMeshesPage";

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
        <main className="flex flex-col w-full h-screen overflow-y-auto">
          <SidebarTrigger />
          <Routes>
            <Route path="/" element={<MainPage />} />
            <Route path="collectors" element={<CollectorMetricsPage />} />
            <Route
              path="collectors/:cluster/:collector"
              element={<CollectorContent />}
            />

            <Route path="victoria" element={<VictoriaPage />} />
            <Route path="victoria/:cluster/:pod" element={<VictoriaDetailsPage />} />

            <Route path="istio/remote-secrets" element={<IstioSecretsPage />} />
            <Route path="istio/meshes" element={<IstioMeshesPage />} />

              {Dashboards.map((d) => (
                <Route path={d.id} element={<DashboardLayout {...d} />}>
                  <Route index element={d.renderList ? d.renderList() : null} />
                  <Route
                    path=":clusterName/:namespace/:objName"
                    element={d.renderDetails ? d.renderDetails() : null}
                  />
                  <Route
                    path=":clusterName/:objName"
                    element={d.renderDetails ? d.renderDetails() : null}
                  />
                </Route>
              ))}

              <Route path="*" element={<NoPage />} />
            </Routes>
          </main>
        </SidebarProvider>
      </BrowserRouter>
  );
}

export default App;
