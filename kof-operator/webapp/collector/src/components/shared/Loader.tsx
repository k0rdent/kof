import { Loader as LoaderIcon } from "lucide-react";
import { JSX } from "react";

const Loader = (): JSX.Element => {
  return (
    <div className="flex w-full h-full justify-center items-center">
      <LoaderIcon className="animate-spin w-8 h-8" />
    </div>
  );
};

export default Loader;
