import { JSX } from "react";
import { Outlet } from "react-router-dom";
import { Separator } from "@/components/generated/ui/separator";
import { useClusterSummariesProvider } from "@/providers/cluster_summaries/ClusterDeploymentsProvider";
import Loader from "@/components/shared/Loader";
import FetchStatus from "@/components/shared/FetchStatus";

const ClusterSummaryLayout = (): JSX.Element => {
  const {
    data: summaries,
    isLoading,
    error,
    fetch,
  } = useClusterSummariesProvider();

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <h1 className="font-bold text-3xl">Cluster Summaries</h1>
      <Separator />
      {isLoading ? (
        <Loader />
      ) : error ? (
        <FetchStatus onReload={fetch}>
          Failed to fetch cluster summaries. Click "Reload" button to try again.
        </FetchStatus>
      ) : !summaries || !summaries.length ? (
        <FetchStatus onReload={fetch}>No cluster summaries found</FetchStatus>
      ) : (
        <Outlet />
      )}
    </div>
  );
};

export default ClusterSummaryLayout;
