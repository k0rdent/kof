import { JSX } from "react";
import { Badge } from "../generated/ui/badge";

interface HealthBadgeProps {
  isHealthy: boolean;
}

const HealthBadge = ({ isHealthy }: HealthBadgeProps): JSX.Element => {
  const badgeColor = isHealthy
    ? "bg-green-50 text-green-700 border-green-200"
    : "bg-red-50 text-red-700 border-red-200";

  const dotColor = isHealthy ? "bg-green-500" : "bg-red-500";

  return (
    <Badge variant="outline" className={badgeColor}>
      <div className={`w-2 h-2 rounded-full mr-1 ${dotColor}`} />
      {isHealthy ? "healthy" : "unhealthy"}
    </Badge>
  );
};

export default HealthBadge;
