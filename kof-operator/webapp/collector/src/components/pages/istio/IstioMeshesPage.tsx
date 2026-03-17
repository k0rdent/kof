import { Button } from "@/components/generated/ui/button";
import { Separator } from "@/components/generated/ui/separator";
import { useIstioMesh } from "@/providers/istio/IstioMeshProvider";
import { JSX } from "react";
import { MeshBody } from "./MeshBody";

const IstioMeshesPage = (): JSX.Element => {
  const { fetch, data, isLoading, error } = useIstioMesh();

  return (
    <div className="flex flex-col w-full h-full p-5 space-y-4 overflow-hidden">
      <header className="flex justify-between items-center">
        <h1 className="font-bold text-3xl">Istio Mesh Topology</h1>
        <Button
          variant="outline"
          size="sm"
          className="cursor-pointer"
          onClick={() => fetch()}
          disabled={isLoading}
        >
          {isLoading ? "Loading…" : "Refresh"}
        </Button>
      </header>
      <Separator />
      <MeshBody isLoading={isLoading} data={data} error={error} onRetry={fetch} />
    </div>
  );
};

export default IstioMeshesPage;
