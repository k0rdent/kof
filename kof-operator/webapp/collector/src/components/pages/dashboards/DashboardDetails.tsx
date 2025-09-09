import { Button } from "@/components/generated/ui/button";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger
} from "@/components/generated/ui/tabs";
import { MoveLeft } from "lucide-react";
import { JSX, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { DashboardData } from "@/components/pages/dashboards/DashboardTypes";
import { K8sObjectSet } from "@/models/k8sObjectSet";
import { K8sObject } from "@/models/k8sObject";
import { capitalizeFirstLetter } from "@/utils/formatter";
import DetailsHeader from "@/components/shared/DetailsHeader";

const DashboardDetails = <
  Items extends K8sObjectSet<Item> | unknown[],
  Item extends K8sObject
>({
  name,
  store,
  detailTabs: details,
  icon: Icon,
  detailsHeaderChild
}: DashboardData<Items, Item>): JSX.Element => {
  const { isLoading, items, selectedItem, selectItem } = store();

  const navigate = useNavigate();
  const { objName } = useParams();

  useEffect(() => {
    if (!isLoading && items && objName) {
      selectItem(objName);
    }
  }, [objName, items, isLoading, selectItem]);

  if (!selectedItem) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <div className="flex flex-col w-full h-full justify-center items-center space-y-4">
          <span>{capitalizeFirstLetter(name)} not found</span>
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
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <DetailsHeader
        icon={Icon}
        title={selectedItem.name}
        isHealthy={selectedItem.isHealthy}
      >
        {detailsHeaderChild ? detailsHeaderChild(selectedItem) : null}
      </DetailsHeader>
      <Tabs defaultValue="status" className="space-y-6">
        <TabsList className="flex w-full">
          {details?.map((d) => {
            const triggerValue = d.triggerValue
              ? d.triggerValue
              : d.name.toLowerCase().replace(" ", "-");

            return d.isDisabledFn && d.isDisabledFn(selectedItem) ? (
              <></>
            ) : (
              <TabsTrigger key={d.name} value={triggerValue}>
                {d.name}
              </TabsTrigger>
            );
          })}
        </TabsList>

        {details?.map((d) => {
          return (
            <TabsContent
              className="max-w-full flex flex-col gap-5"
              value={d.triggerValue}
              key={d.name}
            >
              {d.contentFn(selectedItem)}
            </TabsContent>
          );
        })}
      </Tabs>
    </div>
  );
};

export default DashboardDetails;
