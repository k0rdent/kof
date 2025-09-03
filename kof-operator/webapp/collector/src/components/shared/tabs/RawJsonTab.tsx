import { TabsContent } from "@/components/generated/ui/tabs";
import JsonView from "@uiw/react-json-view";
import { JSX } from "react";

interface RawJsonTabProps {
  object: object;
  // Depth level for collapsing JSON.
  // Starts at 1 for the root object.
  // Use 0 to show JSON fully expanded (no collapse).
  depthLevel?: number;
}

const RawJsonTab = ({
  depthLevel = 2,
  object,
}: RawJsonTabProps): JSX.Element => {
  return (
    <TabsContent value="raw_json" className="flex flex-col gap-5">
      <JsonView
        value={object}
        displayDataTypes={false}
        className="w-full whitespace-normal break-words"
        shouldExpandNodeInitially={(_, props) => {
          return props.level == depthLevel;
        }}
      />
    </TabsContent>
  );
};
export default RawJsonTab;
