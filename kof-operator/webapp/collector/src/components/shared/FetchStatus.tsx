import { JSX } from "react";
import { Button } from "../generated/ui/button";

interface FetchStatusProps {
  children: React.ReactNode;
  onReload?: () => Promise<void>;
}

const FetchStatus = ({ children, onReload }: FetchStatusProps): JSX.Element => {
  return (
    <div className="flex flex-col justify-center items-center mt-[15%]">
      <span className="mb-3">{children}</span>

      {onReload && (
        <Button className="cursor-pointer" onClick={() => onReload()}>
          Reload
        </Button>
      )}
    </div>
  );
};

export default FetchStatus;
