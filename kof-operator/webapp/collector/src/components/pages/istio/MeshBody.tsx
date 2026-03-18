import { Button } from "@/components/generated/ui/button";
import Loader from "@/components/shared/Loader";
import { MeshLink, MeshNode } from "@/providers/istio/IstioMeshProvider";
import { JSX } from "react";
import { MeshGraphView } from "./MeshGraphView";

interface MeshBodyProps {
  isLoading: boolean;
  data: { nodes: MeshNode[]; links: MeshLink[] } | null;
  error: Error | null;
  onRetry: () => void;
}

export const MeshBody = ({
  isLoading,
  data,
  error,
  onRetry,
}: MeshBodyProps): JSX.Element => {
  if (isLoading && !data) return <Loader />;

  if (!isLoading && error) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-4 text-sm">
        <p>Failed to load mesh topology: {error.message}</p>
        <Button className="cursor-pointer" onClick={onRetry}>
          Retry
        </Button>
      </div>
    );
  }

  if (!data || data.nodes.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center">
        No Istio mesh data found. Make sure Istio is installed and remote secrets are
        configured correctly.
      </div>
    );
  }

  return <MeshGraphView data={data} />;
};
