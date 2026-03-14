import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/generated/ui/sheet";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/generated/ui/tabs";
import { ClusterConnectivity } from "@/providers/istio/IstioMeshProvider";
import { JSX } from "react";
import { roleColor } from "./constants";
import { EndpointsTab } from "./EndpointsTab";
import { InfoTab } from "./InfoTab";
import { GraphNode } from "./types";

interface ClusterDetailSheetProps {
  open: boolean;
  node: GraphNode | null;
  connectivity: ClusterConnectivity | null;
  isLoading: boolean;
  error: Error | null;
  onOpenChange: (open: boolean) => void;
  onRefresh: () => void;
}

export const ClusterDetailSheet = ({
  open,
  node,
  connectivity,
  isLoading,
  error,
  onOpenChange,
  onRefresh,
}: ClusterDetailSheetProps): JSX.Element => (
  <Sheet open={open} onOpenChange={onOpenChange}>
    <SheetContent
      side="right"
      className="sm:max-w-200 flex flex-col gap-0 p-0 bg-slate-900 border-slate-700"
    >
      {node && (
        <>
          <SheetHeader className=" border-b border-slate-700">
            <SheetTitle className="flex items-center gap-2 text-white text-lg font-semibold truncate">
              <span
                className="inline-block w-3 h-3 rounded-full shrink-0"
                style={{ backgroundColor: roleColor(node.role) }}
              />
              {node.name}
            </SheetTitle>
          </SheetHeader>

          <Tabs defaultValue="endpoints" className="flex flex-col flex-1 min-h-0">
            <TabsList className="mx-6 mt-4 mb-2 grid grid-cols-2 bg-slate-800">
              <TabsTrigger value="endpoints">Endpoints</TabsTrigger>
              <TabsTrigger value="info">Info</TabsTrigger>
            </TabsList>

            <TabsContent
              value="endpoints"
              className="flex-1 overflow-hidden flex flex-col px-6 pb-6"
            >
              <EndpointsTab
                clusterName={node.id}
                connectivity={connectivity}
                isLoading={isLoading}
                error={error}
                onRefresh={onRefresh}
              />
            </TabsContent>

            <TabsContent value="info" className="flex-1 overflow-auto px-6 pb-6">
              <InfoTab node={node} />
            </TabsContent>
          </Tabs>
        </>
      )}
    </SheetContent>
  </Sheet>
);
