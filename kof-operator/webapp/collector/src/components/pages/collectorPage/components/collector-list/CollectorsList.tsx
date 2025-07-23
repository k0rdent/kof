import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import { JSX } from "react";
import CollectorsTable from "./CollectorsTable";
import { Loader } from "lucide-react";
import { Button } from "@/components/generated/ui/button";

const CollectorsList = (): JSX.Element => {
  const { data, isLoading, error, fetch } = useCollectorMetricsState();

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

  return (
    <div className="flex flex-col space-y-8">
      {data?.clusters.map((cluster) => (
        <CollectorsTable key={cluster.name} cluster={cluster} />
      ))}
    </div>
  );
};

export default CollectorsList;
