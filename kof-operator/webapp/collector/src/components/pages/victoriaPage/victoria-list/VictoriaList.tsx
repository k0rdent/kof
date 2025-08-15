import { Button } from "@/components/generated/ui/button";
import { useVictoriaMetricsState } from "@/providers/victoria_metrics/VictoriaMetricsProvider";
import { Loader } from "lucide-react";
import { JSX } from "react";
import VictoriaTable from "./VictoriaTable";

const VictoriaList = (): JSX.Element => {
  const { data, isLoading, error, fetch } = useVictoriaMetricsState();

  if (!isLoading && error) {
    return (
      <div className="flex flex-col justify-center items-center mt-[15%]">
        <span className="mb-3">
          Failed to fetch collectors metrics. Click "Reload" button to try
          again.
        </span>
        <Button className="cursor-pointer" onClick={() => fetch()}>
          Reload
        </Button>
      </div>
    );
  }

  if (isLoading && !data) {
    return (
      <div className="flex w-full justify-center items-center mt-[15%]">
        <Loader className="animate-spin w-8 h-8"></Loader>
      </div>
    );
  }

  if (!isLoading && !data) {
    return (
      <div className="flex w-full h-screen justify-center items-center">
        No clusters found
      </div>
    );
  }

  return (
    <div className="flex flex-col space-y-8">
      {data?.clusters.map((cluster) => (
        <VictoriaTable key={cluster.name} cluster={cluster} />
      ))}
    </div>
  );
};

export default VictoriaList;
