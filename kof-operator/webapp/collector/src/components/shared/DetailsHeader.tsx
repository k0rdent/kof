import {
  ForwardRefExoticComponent,
  JSX,
  ReactNode,
  RefAttributes,
} from "react";
import { Button } from "../generated/ui/button";
import { LucideProps, MoveLeft } from "lucide-react";
import HealthBadge from "./HealthBadge";
import { useNavigate } from "react-router-dom";

interface DetailsHeaderProps {
  icon: ForwardRefExoticComponent<
    Omit<LucideProps, "ref"> & RefAttributes<SVGSVGElement>
  >;
  title: string;
  isHealthy: boolean;
  children?: ReactNode;
}

const DetailsHeader = ({
  children,
  icon: Icon,
  title,
  isHealthy,
}: DetailsHeaderProps): JSX.Element => {
  const navigate = useNavigate();

  return (
    <div className="flex flex-col space-y-6">
      <div className="flex items-center space-x-6">
        <Button
          variant="outline"
          className="cursor-pointer"
          onClick={() => {
            navigate(-1);
          }}
        >
          <MoveLeft />
          <span>Back to Table</span>
        </Button>

        {children}
      </div>
      <div className="flex gap-4 items-center mb-2">
        <Icon />
        <span className="font-bold text-xl">{title}</span>
        <HealthBadge isHealthy={isHealthy} />
      </div>
    </div>
  );
};

export default DetailsHeader;
