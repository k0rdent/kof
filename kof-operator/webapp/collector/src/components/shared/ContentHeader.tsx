import { MoveLeft, Server } from "lucide-react";
import { JSX } from "react";
import HealthBadge from "@/components/shared/HealthBadge";
import { Button } from "@/components/generated/ui/button";
import { useNavigate } from "react-router-dom";
import { Pod } from "../pages/collectorPage/models";
import { StoreApi, UseBoundStore } from "zustand";
import { DefaultProviderState } from "@/providers/DefaultProviderState";

interface ContentHeaderProps {
  tableURL: string;
  title: string;
  pod: Pod;
  state: UseBoundStore<StoreApi<DefaultProviderState>>;
}

const ContentHeader = ({
  tableURL,
  title,
  pod,
  state,
}: ContentHeaderProps): JSX.Element => {
  const navigate = useNavigate();
  const { selectedPod: selectedCollector } = state();

  if (!selectedCollector) {
    return <></>;
  }

  return (
    <div className="space-y-6">
      <Button
        variant="outline"
        className="cursor-pointer"
        onClick={() => {
          navigate(tableURL);
        }}
      >
        <MoveLeft />
        <span>Back to Table</span>
      </Button>
      <div className="flex items-center gap-3 mb-4">
        <Server className="w-5 h-5"></Server>
        <h1 className="font-bold text-xl">
          {title}: {pod.name}
        </h1>
        <HealthBadge isHealthy={pod.isHealthy} />
      </div>
    </div>
  );
};

export default ContentHeader;
