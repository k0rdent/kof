import { JSX } from "react";

interface StatRowProps {
  text: string;
  textStyles?: string;
  value: string | number;
  valueStyles?: string;
  containerStyle?: string;
}

const StatRow = ({
  text,
  value,
  textStyles,
  valueStyles,
  containerStyle,
}: StatRowProps): JSX.Element => {
  return (
    <div className={`flex justify-between ${containerStyle}`}>
      <span className={`text-sm ${textStyles}`}>{text}</span>
      <span className={`font-medium ${valueStyles}`}>{value}</span>
    </div>
  );
};

export default StatRow;
