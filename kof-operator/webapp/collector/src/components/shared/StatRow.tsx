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
  hint?: string;
}

const StatRow = ({
  text,
  value,
  hint,
  textStyles,
  valueStyles,
  containerStyle,
}: StatRowProps): JSX.Element => {
  return (
    <div className={`flex justify-between ${containerStyle}`}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className={`text-sm cursor-default ${textStyles}`}>{text}</span>
        </TooltipTrigger>
        {hint && (
          <TooltipContent sideOffset={-6}>
            <p>{hint}</p>
          </TooltipContent>
        )}
      </Tooltip>
      <span className={`font-medium ${valueStyles}`}>{value}</span>
    </div>
  );
};

export default StatRow;
