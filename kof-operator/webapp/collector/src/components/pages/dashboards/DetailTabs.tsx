import { JSX } from "react";
import { DetailsTab } from "./DashboardTypes";
import { K8sObject } from "@/models/k8sObject";
import StatusTab from "@/components/shared/tabs/StatusTab";
import RawJsonTab from "@/components/shared/tabs/RawJsonTab";
import MetadataTab from "@/components/shared/tabs/MetadataTab";

export const DetailMetadataTab = <Item extends K8sObject>(): DetailsTab<Item> => {
  return DetailsNewTab("Metadata", (item: Item) => (
    <MetadataTab metadata={item.metadata} />
  ));
};

export const DetailRawJsonTab = <Item extends K8sObject>(
  depthLevel: number = 4
): DetailsTab<Item> => {
  return DetailsNewTab("Raw Json", (item: Item) => (
    <RawJsonTab depthLevel={depthLevel} object={item.raw} />
  ));
};

export const DetailStatusTab = <Item extends K8sObject>(): DetailsTab<Item> => {
  return DetailsNewTab("Status", (item: Item) => (
    <StatusTab conditions={item.status.conditions} />
  ));
};

export const DetailsNewTab = <Item extends K8sObject>(
  name: string,
  contentFn: (item: Item) => JSX.Element,
  isDisabledFn?: (item: Item) => boolean,
  triggerValue?: string
): DetailsTab<Item> => {
  return {
    name,
    contentFn,
    isDisabledFn,
    triggerValue: triggerValue ?? name.toLowerCase().replace(" ", "-")
  };
};
