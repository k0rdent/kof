import { MoveLeft, Server } from "lucide-react";
import { JSX } from "react";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";
import HealthBadge from "@/components/shared/HealthBadge";
import { Button } from "@/components/generated/ui/button";
import { useNavigate } from "react-router-dom";

const CollectorContentHeader = (): JSX.Element => {
  const navigate = useNavigate();
  const { selectedCollector } = useCollectorMetricsState();

  if (!selectedCollector) {
    return <></>;
  }

  return (
    <div className="space-y-6">
      <Button
        variant="outline"
        className="cursor-pointer"
        onClick={() => {
          navigate("/collectors");
        }}
      >
        <MoveLeft />
        <span>Back to Table</span>
      </Button>
      <div className="flex items-center gap-3 mb-4">
        <Server className="w-5 h-5"></Server>
        <h1 className="font-bold text-xl">
          Collector: {selectedCollector.name}
        </h1>
        <HealthBadge isHealthy={selectedCollector.isHealthy} />
      </div>
    </div>
  );
};

export default CollectorContentHeader;
