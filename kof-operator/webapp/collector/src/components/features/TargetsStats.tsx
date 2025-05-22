import { JSX } from "react";
import { Cluster, Target } from "@/models/PrometheusTarget";
import { Badge } from "../ui/badge";
import { getTargets } from "@/utils/cluster";

interface TargetStatsProps {
  title?: string;
  clusters: Cluster[];
}

const TargetStats = ({ title, clusters }: TargetStatsProps): JSX.Element => {
  const targets: Target[] = getTargets(clusters);

  return (
    <div className="flex items-end gap-5">
      <h1 className="text-2xl font-bold text-gray-800">{title}</h1>
      <p className="text-lg font-semibold text-gray-500">
        <Badge className="bg-amber-300 text-black  mr-2">
          {targets.filter((t) => t.health === "unknown").length} Unknown
        </Badge>
        •
        <Badge className="bg-red-500 ml-2 mr-2">
          {targets.filter((t) => t.health === "down").length} Down
        </Badge>
        •
        <Badge className="bg-green-500 ml-2 mr-2">
          {targets.filter((t) => t.health === "up").length} Up
        </Badge>
      </p>
    </div>
  );
};

export default TargetStats;
