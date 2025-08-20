import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/generated/ui/alert";
import { METRICS } from "@/constants/metrics.constants";
import { TriangleAlert } from "lucide-react";
import { JSX } from "react";
import { Pod } from "../pages/collectorPage/models";

const UnhealthyAlert = ({ pod }: { pod: Pod }): JSX.Element => {
  if (!pod || pod.isHealthy) {
    return <></>;
  }

  const alertMessage = pod.getMetric(METRICS.CONDITION_READY_MESSAGE.name)
    ?.metricValues[0].value;
  const alertReason = pod.getMetric(METRICS.CONDITION_READY_REASON.name)
    ?.metricValues[0].value;

  return (
    <Alert variant="destructive">
      <TriangleAlert />
      <AlertTitle>Pod is unhealthy</AlertTitle>
      <AlertDescription className="gap-0">
        <p>Reason: {alertReason}</p>
        <div>Message: {alertMessage}</div>
      </AlertDescription>
    </Alert>
  );
};

export default UnhealthyAlert;
