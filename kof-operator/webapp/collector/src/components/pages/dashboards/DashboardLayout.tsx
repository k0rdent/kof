import { JSX } from "react";
import { Outlet } from "react-router-dom";
import { Separator } from "@/components/generated/ui/separator";
import { AllDashboards } from "./DashboardFactories";
import Loader from "@/components/shared/Loader";
import FetchStatus from "@/components/shared/FetchStatus";

const DashboardLayout = ({ name, store }: AllDashboards): JSX.Element => {
  const { items, isLoading, error, fetch } = store();
  const lowerCaseName: string = name.toLowerCase();

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <h1 className="font-bold text-3xl">{name}</h1>
      <Separator />
      {isLoading ? (
        <Loader />
      ) : error ? (
        <FetchStatus onReload={fetch}>
          Failed to fetch {lowerCaseName}. Click "Reload" button to try again.
        </FetchStatus>
      ) : !items || !items.length ? (
        <FetchStatus onReload={fetch}>No {lowerCaseName} found</FetchStatus>
      ) : (
        <Outlet />
      )}
    </div>
  );
};

export default DashboardLayout;
