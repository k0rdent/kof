import { JSX } from "react";
import { Separator } from "@/components/generated/ui/separator";
import CollectorsList from "./components/collector-list/CollectorsList";
import CollectorsPageHeader from "./CollectorsPageHeader";

const CollectorMetricsPage = (): JSX.Element => {
  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <CollectorsPageHeader />
      <Separator />
      <CollectorsList />
    </div>
  );
};

export default CollectorMetricsPage;
