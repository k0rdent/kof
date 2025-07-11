import { Badge } from "@/components/generated/ui/badge";
import { Server } from "lucide-react";
import { JSX } from "react";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";

const CollectorContentHeader = (): JSX.Element => {
  const { selectedCollector } = useCollectorMetricsState();

  if (!selectedCollector) {
    return <></>;
  }

  const badgeColor = selectedCollector.isHealthy
    ? "bg-green-50 text-green-700 border-green-200"
    : "bg-red-50 text-red-700 border-red-200";

  return (
    <div className="flex items-center gap-3">
      <Server className="w-5 h-5"></Server>
      <h1 className="font-bold text-xl">Collector: {selectedCollector.name}</h1>
      <Badge variant="outline" className={badgeColor}>
        {selectedCollector.isHealthy ? "healthy" : "unhealthy"}
      </Badge>
    </div>
  );
};

export default CollectorContentHeader;
