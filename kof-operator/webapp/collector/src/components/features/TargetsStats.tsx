import { JSX } from "react";
import { Target } from "@/models/PrometheusTarget";
import { Badge } from "../ui/badge";
import { Cluster } from "@/models/Cluster";
import { getTargetCountByHealth } from "@/utils/target";

interface TargetStatsProps {
  clusters: Cluster[];
}

const TargetStats = ({ clusters }: TargetStatsProps): JSX.Element => {
  const targets: Target[] = clusters.flatMap((cluster) => cluster.targets);

  return (
    <div className="flex items-end gap-5">
      <p className="text-lg font-semibold text-gray-500">
        <Badge className="bg-amber-300 text-black mr-2">
          {getTargetCountByHealth(targets, "unknown")} Unknown
        </Badge>
        •
        <Badge className="bg-red-500 ml-2 mr-2">
          {getTargetCountByHealth(targets, "down")} Down
        </Badge>
        •
        <Badge className="bg-green-500 ml-2 mr-2">
          {getTargetCountByHealth(targets, "up")} Up
        </Badge>
      </p>
    </div>
  );
};

export default TargetStats;
