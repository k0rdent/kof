import { PrometheusContext } from "@/models/PrometheusTarget";
import { createContext } from "react";

const PrometheusTargetsContext = createContext<PrometheusContext | undefined>(
  undefined
);

export default PrometheusTargetsContext;
