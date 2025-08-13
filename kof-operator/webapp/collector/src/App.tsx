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
import CollectorContent from "./components/pages/collectorPage/components/collector-details/CollectorContent";
import VictoriaPage from "./components/pages/victoriaPage/VictoriaPage";
import { useVictoriaMetricsState } from "./providers/victoria_metrics/VictoriaMetricsProvider";
import VictoriaDetailsPage from "./components/pages/victoriaPage/victoria-details/VictoriaDetailsPage";

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
            <Route path="*" element={<NoPage />} />
          </Routes>
        </main>
      </SidebarProvider>
    </BrowserRouter>
  );
}

export default App;
