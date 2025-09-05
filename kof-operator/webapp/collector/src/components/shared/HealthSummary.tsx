import { JSX } from "react";

export interface HealthSummaryProps {
  totalCount: number;
  healthyCount: number;
  unhealthyCount: number;
  totalText?: string;
  className?: string;
}

const HealthSummary = ({
  totalCount,
  healthyCount,
  unhealthyCount,
  totalText = "Total",
  className = "",
}: HealthSummaryProps): JSX.Element => {
  return (
    <div className={`flex gap-2 ${className}`}>
      <span>{`${totalCount} ${totalText}`}</span>
      <span>•</span>
      <span className="text-green-600">{healthyCount} healthy</span>
      <span>•</span>
      <span className="text-red-600">{unhealthyCount} unhealthy</span>
    </div>
  );
};

export default HealthSummary;
