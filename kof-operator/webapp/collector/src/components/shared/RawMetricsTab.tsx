import { TabsContent } from "@/components/generated/ui/tabs";
import { DefaultProviderState } from "@/providers/DefaultProviderState";
import JsonView from "@uiw/react-json-view";
import { JSX } from "react";
import { StoreApi, UseBoundStore } from "zustand";

interface RawMetricsTabProps {
  state: UseBoundStore<StoreApi<DefaultProviderState>>;
}

// Level enum for a map of metrics
const LEVEL = Object.freeze({
  ROOT: 1, // Top-level object
  METRICS_MAP: 2, // The metrics map
  VALUES_ARRAY: 3, // The values array
});

const RawMetricsTab = ({ state }: RawMetricsTabProps): JSX.Element => {
  const { selectedPod } = state();

  return (
    <TabsContent value="raw_metrics" className="flex flex-col gap-5">
      <JsonView
        value={selectedPod?.getMetrics()}
        displayDataTypes={false}
        className="w-full whitespace-normal break-words"
        shouldExpandNodeInitially={(_, props) => {
          return props.level == LEVEL.METRICS_MAP;
        }}
      />
    </TabsContent>
  );
};
export default RawMetricsTab;
