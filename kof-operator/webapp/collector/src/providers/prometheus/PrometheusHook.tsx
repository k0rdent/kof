import { useContext } from "react";
import PrometheusTargetsContext from "./PrometheusContext";

const usePrometheusTarget = () => {
  return useContext(PrometheusTargetsContext);
};

export default usePrometheusTarget
