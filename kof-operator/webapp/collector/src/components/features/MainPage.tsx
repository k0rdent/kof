import { JSX } from "react";
import TargetList from "./TargetsList";
import TargetFilter from "./TargetFilter";
import { Separator } from "../ui/separator";

const MainPage = (): JSX.Element => {
  return (
    <div className="w-100% bg-white">
      <TargetFilter></TargetFilter>
      <Separator />
      <TargetList></TargetList>
    </div>
  );
};

export default MainPage;
