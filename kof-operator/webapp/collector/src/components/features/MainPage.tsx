import { JSX } from "react";
import TargetList from "./TargetsList";
import TargetFilter from "./TargetFilter";
import { Separator } from "@/components/generated/ui/separator";
import PrometheusTargetProvider from "@/providers/prometheus/PrometheusTargetsProvider";

const MainPage = (): JSX.Element => {
  return (
    <PrometheusTargetProvider>
      <div className="w-full bg-white">
        <TargetFilter></TargetFilter>
        <Separator />
        <TargetList></TargetList>
      </div>
    </PrometheusTargetProvider>
  );
};

export default MainPage;
