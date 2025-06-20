import MainPage from "./components/features/MainPage";
import PrometheusTargetProvider from "./providers/prometheus/PrometheusTargetsProvider";

function App() {
  return (
    <PrometheusTargetProvider>
      <main className="w-full h-full min-h-screen min-w-screen">
        <MainPage></MainPage>
      </main>
    </PrometheusTargetProvider>
  );
}

export default App;
