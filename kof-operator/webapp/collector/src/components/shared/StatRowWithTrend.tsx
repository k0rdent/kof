import { Trend } from "@/providers/collectors_metrics/CollectorsMetricsRecordManager";
import { TrendingUp } from "lucide-react";
import { JSX } from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "../generated/ui/tooltip";

interface StatRowProps {
  text: string;
  textStyles?: string;
  value: string | number;
  valueStyles?: string;
  containerStyle?: string;
  trend: Trend;
  isPositiveTrend?: boolean;
  hint?: string;
}

const StatRowWithTrend = ({
  text,
  value,
  textStyles,
  valueStyles,
  containerStyle,
  trend,
  hint,
  isPositiveTrend = true,
}: StatRowProps): JSX.Element => {
  const isTrendGood = trend?.isTrending === isPositiveTrend;
  const trendMessageColor = isTrendGood ? "text-green-600" : "text-red-600";

  return (
    <div>
      <div className={`flex justify-between mb-0 ${containerStyle}`}>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className={`text-sm cursor-default ${textStyles}`}>
              {text}
            </span>
          </TooltipTrigger>
          {hint && (
            <TooltipContent sideOffset={-6}>
              <p>{hint}</p>
            </TooltipContent>
          )}
        </Tooltip>

        <div
          className={`flex gap-2 items-center font-medium ${trendMessageColor}`}
        >
          {trend.isTrending && <TrendingUp className="w-5 h-5" />}
          {trend.message}
        </div>
      </div>
      <div className="flex justify-end">
        <span className={`text-sm ${valueStyles}`}>{value}</span>
      </div>
    </div>
  );
};
export default StatRowWithTrend;
