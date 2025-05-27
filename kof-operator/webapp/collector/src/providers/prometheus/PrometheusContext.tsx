import { PrometheusContext } from "@/models/PrometheusTarget";
import { createContext } from "react";
import { FilterFunction } from "./PrometheusTargetsProvider";

const PrometheusTargetsContext = createContext<PrometheusContext>({
  data: null,
  filteredData: null,
  loading: false,
  error: null,
  addFilter: function (name: string, filterFn: FilterFunction): string {
    console.log(name, filterFn)
    throw new Error("Function not implemented.");
  },
  removeFilter: function (id: string): void {
    throw new Error(`Function not implemented. ${id}`);
  },
  clearFilters: function (): void {
    throw new Error("Function not implemented.");
  },
  fetchPrometheusTargets: function (): Promise<void> {
    throw new Error("Function not implemented.");
  },
});

export default PrometheusTargetsContext;
