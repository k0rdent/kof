import { Separator } from "@/components/generated/ui/separator";
import { JSX } from "react";
import VictoriaPageHeader from "./VictoriaPageHeader";
import VictoriaList from "./victoria-list/VictoriaList";

const VictoriaPage = (): JSX.Element => {
  return (
    <div className="flex flex-col w-full h-full p-5 space-y-8">
      <VictoriaPageHeader />
      <Separator />
      <VictoriaList />
    </div>
  );
};

export default VictoriaPage;
