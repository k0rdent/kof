import { JSX } from "react";
import { Outlet } from "react-router-dom";
import { Separator } from "@/components/generated/ui/separator";
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import Loader from "@/components/shared/Loader";
import { Button } from "@/components/generated/ui/button";

const ClusterDeploymentLayout = (): JSX.Element => {
  const { data: clusters, isLoading, error } = useClusterDeploymentsProvider();

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <h1 className="font-bold text-3xl">Cluster Deployments</h1>
      <Separator />
      {isLoading ? <Loader />
        : error ? <FetchError />
        : !clusters || !clusters.length ? <EmptyState />
        : <Outlet />}
    </div>
  );
};

export default ClusterDeploymentLayout;

const FetchError = (): JSX.Element => {
  const { fetch } = useClusterDeploymentsProvider();
  return (
    <div className="flex flex-col justify-center items-center mt-[15%]">
      <span className="mb-3">
        Failed to fetch cluster deployments. Click "Reload" button to try again.
      </span>
      <Button className="cursor-pointer" onClick={() => fetch()}>
        Reload
      </Button>
    </div>
  );
};

const EmptyState = (): JSX.Element => (
  <div className="flex w-full h-full justify-center items-center">
    No cluster deployments found
  </div>
);
