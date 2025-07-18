import { Alert, AlertDescription, AlertTitle } from "@/components/generated/ui/alert";
import { METRICS } from "@/constants/metrics.constants";
import { TriangleAlert } from "lucide-react";
import { JSX } from "react";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";

const UnhealthyAlert = (): JSX.Element => {
  const { selectedCollector } = useCollectorMetricsState();

  if (!selectedCollector || selectedCollector.isHealthy) {
    return <></>;
  }

  const alertMessage = selectedCollector.getStringMetric(
    METRICS.OTELCOL_CONDITION_READY_MESSAGE
  );
  const alertReason = selectedCollector.getStringMetric(
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
