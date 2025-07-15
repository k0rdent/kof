import { Trend } from "@/providers/collectors_metrics/CollectorsMetricsRecordManager";
import { TrendingUp } from "lucide-react";
import { JSX } from "react";

interface StatRowProps {
  text: string;
  textStyles?: string;
  value: string | number;
  valueStyles?: string;
  containerStyle?: string;
  trend: Trend | null;
  isPositiveTrend?: boolean;
}

const StatRowWithTrend = ({
  text,
  value,
  textStyles,
  valueStyles,
  containerStyle,
  trend,
  isPositiveTrend = true,
}: StatRowProps): JSX.Element => {
  const isTrendGood = trend?.isTrending === isPositiveTrend;
  const trendMessageColor = isTrendGood ? "text-green-600" : "text-red-600";

  return (
    <div>
      <div className={`flex justify-between mb-0 ${containerStyle}`}>
        <span className={`text-sm ${textStyles}`}>{text}</span>
        {trend && (
          <div
            className={`flex gap-2 items-center font-medium ${trendMessageColor}`}
          >
            {trend.isTrending && <TrendingUp className="w-5 h-5" />}
            {trend.message}
          </div>
        )}
      </div>
      <div className="flex justify-end">
        <span className={`text-sm ${valueStyles}`}>{value}</span>
      </div>
    </div>
  );
};
export default StatRowWithTrend;
