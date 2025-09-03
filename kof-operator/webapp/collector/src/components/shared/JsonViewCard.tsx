import { LucideProps } from "lucide-react";
import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";
import { MetricRow, MetricsCard } from "./MetricsCard";
import JsonView from "@uiw/react-json-view";

interface JsonViewCardProps {
  title: string;
  icon: ForwardRefExoticComponent<
    Omit<LucideProps, "ref"> & RefAttributes<SVGSVGElement>
  >;
  data: object;
  shortenTextAfterLength?: number;
}

const JsonViewCard = ({
  title,
  icon,
  data,
  shortenTextAfterLength = 0,
}: JsonViewCardProps): JSX.Element => {
  const rows: MetricRow[] = [
    {
      title: "",
      customRow: () => (
        <div className="flex flex-col gap-2 w-full">
          <JsonView
            value={data}
            displayDataTypes={false}
            className="w-full whitespace-normal break-words"
            shortenTextAfterLength={shortenTextAfterLength}
          />
        </div>
      ),
    },
  ];

  return <MetricsCard rows={rows} icon={icon} title={title} />;
};

export default JsonViewCard;
