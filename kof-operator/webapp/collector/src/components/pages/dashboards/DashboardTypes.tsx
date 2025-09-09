import { ProviderStore } from "@/providers/ProviderAbstract";
import { LucideProps } from "lucide-react";
import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";
import { StoreApi, UseBoundStore } from "zustand";
import { CustomizedTableHeadProps } from "../collectorPage/components/collector-list/CollectorTableHead";

export interface DetailsTab<Item> {
  name: string;
  triggerValue: string;
  contentFn: (item: Item) => JSX.Element;
  isDisabledFn?: (item: Item) => boolean;
}

export interface TableColumn<Item> {
  head: CustomizedTableHeadProps;
  valueFn: (item: Item) => JSX.Element;
}

export interface DashboardData<Items, Item> {
  name: string;
  id: string;
  store: UseBoundStore<StoreApi<ProviderStore<Items, Item>>>;
  icon: ForwardRefExoticComponent<
    Omit<LucideProps, "ref"> & RefAttributes<SVGSVGElement>
  >;
  tableCols: TableColumn<Item>[];
  detailTabs: DetailsTab<Item>[];
  detailsHeaderChild?: (item: Item) => JSX.Element;
  renderDetails?: () => JSX.Element;
  renderList?: () => JSX.Element;
}
