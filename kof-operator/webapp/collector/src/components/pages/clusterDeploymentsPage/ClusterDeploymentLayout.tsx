import { JSX } from "react";
import { Outlet } from "react-router-dom";
import { Separator } from "@/components/generated/ui/separator";
import { useClusterDeploymentsProvider } from "@/providers/ClusterDeploymentsProvider";
import Loader from "@/components/shared/Loader";
import FetchStatus from "@/components/shared/FetchStatus";

const ClusterDeploymentLayout = (): JSX.Element => {
  const {
    items: clusters,
    isLoading,
    error,
    fetch,
  } = useClusterDeploymentsProvider();

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <h1 className="font-bold text-3xl">Cluster Deployments</h1>
      <Separator />
      {isLoading ? (
        <Loader />
      ) : error ? (
        <FetchStatus onReload={fetch}>
          Failed to fetch cluster deployments. Click "Reload" button to try
          again.
        </FetchStatus>
      ) : !clusters || !clusters.length ? (
        <FetchStatus onReload={fetch}>No cluster deployments found</FetchStatus>
      ) : (
        <Outlet />
      )}
    </div>
  );
};

export default ClusterDeploymentLayout;
