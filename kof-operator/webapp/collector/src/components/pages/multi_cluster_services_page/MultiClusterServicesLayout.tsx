import { JSX } from "react";
import { Outlet } from "react-router-dom";
import { Separator } from "@/components/generated/ui/separator";
import { useClusterSummariesProvider } from "@/providers/ClusterSummariesProvider";
import Loader from "@/components/shared/Loader";
import FetchStatus from "@/components/shared/FetchStatus";

const MultiClusterServicesLayout = (): JSX.Element => {
  const {
    items: services,
    isLoading,
    error,
    fetch,
  } = useClusterSummariesProvider();

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <h1 className="font-bold text-3xl">Multi Cluster Services</h1>
      <Separator />
      {isLoading ? (
        <Loader />
      ) : error ? (
        <FetchStatus onReload={fetch}>
          Failed to fetch multi cluster services. Click "Reload" button to try again.
        </FetchStatus>
      ) : !services || !services.length ? (
        <FetchStatus onReload={fetch}>No multi cluster services found</FetchStatus>
      ) : (
        <Outlet />
      )}
    </div>
  );
};

export default MultiClusterServicesLayout;
