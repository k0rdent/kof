import AppSidebar from "./components/features/AppSidebar";
import MainPage from "./components/features/MainPage";
import CollectorMetricsPage from "./components/pages/collectorPage/CollectorPage";
import NoPage from "./components/pages/NoPage";
import { SidebarProvider, SidebarTrigger } from "@/components/generated/ui/sidebar";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { useCollectorMetricsState } from "./providers/collectors_metrics/CollectorsMetricsProvider";
import { useEffect } from "react";

function App() {
  const { fetch } = useCollectorMetricsState();
  useEffect(() => {
    const intervalId = setInterval(() => {
      fetch(true);
    }, 20 * 1000);

    return () => clearInterval(intervalId);
  }, [fetch]);

  return (
    <BrowserRouter>
      <SidebarProvider>
        <AppSidebar />
        <main className="flex flex-col w-full h-full">
          <SidebarTrigger />
          <Routes>
            <Route path="/" element={<MainPage />}></Route>
            <Route path="collectors" element={<CollectorMetricsPage />}></Route>
            <Route path="*" element={<NoPage />} />
          </Routes>
        </main>
      </SidebarProvider>
    </BrowserRouter>
  );
}

export default App;
