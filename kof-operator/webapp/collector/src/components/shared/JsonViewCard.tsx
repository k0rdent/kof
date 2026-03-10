import { LucideProps } from "lucide-react";
import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";
import { MetricRow, MetricsCard } from "./MetricsCard";
import CustomJsonView from "./tabs/RawJsonTab";

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
          <CustomJsonView
            object={data}
            shortenTextAfterLength={shortenTextAfterLength}
          />
        </div>
      ),
    },
  ];

  return <MetricsCard rows={rows} icon={icon} title={title} />;
};

export default JsonViewCard;
