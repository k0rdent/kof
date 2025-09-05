import { JSX } from "react";
import { Outlet } from "react-router-dom";
import { Separator } from "@/components/generated/ui/separator";
import { useStateManagementProvidersProvider } from "@/providers/StateManagementProvidersProvider";
import Loader from "@/components/shared/Loader";
import FetchStatus from "@/components/shared/FetchStatus";

const StateManagementProviderLayout = (): JSX.Element => {
  const {
    items: providers,
    isLoading,
    error,
    fetch,
  } = useStateManagementProvidersProvider();

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <h1 className="font-bold text-3xl">State Management Provider</h1>
      <Separator />
      {isLoading ? (
        <Loader />
      ) : error ? (
        <FetchStatus onReload={fetch}>
          Failed to fetch state management provider. Click "Reload" button to
          try again.
        </FetchStatus>
      ) : !providers || !providers.length ? (
        <FetchStatus onReload={fetch}>
          No state management provider found
        </FetchStatus>
      ) : (
        <Outlet />
      )}
    </div>
  );
};

export default StateManagementProviderLayout;
