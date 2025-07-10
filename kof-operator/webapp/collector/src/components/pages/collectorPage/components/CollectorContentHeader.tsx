import { Badge } from "@/components/ui/badge";
import { Server } from "lucide-react";
import { JSX } from "react";
import { Pod } from "../models";

const CollectorContentHeader = ({
  collector,
}: {
  collector: Pod;
}): JSX.Element => {
  const badgeColor = collector.isHealthy
    ? "bg-green-50 text-green-700 border-green-200"
    : "bg-red-50 text-red-700 border-red-200";

  return (
    <div className="flex items-center gap-3">
      <Server className="w-5 h-5"></Server>
      <h1 className="font-bold text-xl">Collector: {collector.name}</h1>
      <Badge variant="outline" className={badgeColor}>
        {collector.isHealthy ? "healthy" : "unhealthy"}
      </Badge>
    </div>
  );
};

export default CollectorContentHeader;
