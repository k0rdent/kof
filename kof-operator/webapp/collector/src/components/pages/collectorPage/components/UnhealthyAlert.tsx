import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { METRICS } from "@/constants/metrics.constants";
import { TriangleAlert } from "lucide-react";
import { JSX } from "react";
import { Pod } from "../models";

const UnhealthyAlert = ({ collector }: { collector: Pod }): JSX.Element => {
  const alertMessage = collector.getStringMetric(
    METRICS.OTELCOL_CONDITION_READY_MESSAGE
  );
  const alertReason = collector.getStringMetric(
    METRICS.OTELCOL_CONDITION_READY_REASON
  );

  return (
    <Alert variant="destructive">
      <TriangleAlert />
      <AlertTitle>Collector is unhealthy</AlertTitle>
      <AlertDescription className="gap-0">
        <p>Reason: {alertReason}</p>
        <div>Message: {alertMessage}</div>
      </AlertDescription>
    </Alert>
  );
};

export default UnhealthyAlert;
