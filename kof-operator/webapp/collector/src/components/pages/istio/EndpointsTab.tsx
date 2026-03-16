import { Button } from "@/components/generated/ui/button";
import { Input } from "@/components/generated/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/generated/ui/table";
import Loader from "@/components/shared/Loader";
import { ClusterConnectivity } from "@/providers/istio/IstioMeshProvider";
import { RefreshCw, Search } from "lucide-react";
import { JSX, useState } from "react";

interface EndpointsTabProps {
  clusterName: string;
  connectivity: ClusterConnectivity | null;
  isLoading: boolean;
  error: Error | null;
  onRefresh: () => void;
}

export const EndpointsTab = ({
  connectivity,
  isLoading,
  error,
  onRefresh,
}: EndpointsTabProps): JSX.Element => {
  const [search, setSearch] = useState("");

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 text-sm text-slate-400">
        <p>Failed to load endpoints: {error.message}</p>
        <Button
          size="sm"
          variant="outline"
          className="cursor-pointer"
          onClick={onRefresh}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (connectivity === null) {
    return (
      <div className="flex flex-1 items-center justify-center text-slate-400 text-sm">
        Select a cluster to load endpoints.
      </div>
    );
  }

  const { connectedClusters } = connectivity;

  const query = search.trim().toLowerCase();

  const filteredClusters = connectedClusters
    .map((rc) => ({
      ...rc,
      services: query
        ? rc.services.filter(
            (svc) =>
              rc.clusterId.toLowerCase().includes(query) ||
              svc.serviceFqdn.toLowerCase().includes(query) ||
              (svc.workloadName ?? "").toLowerCase().includes(query) ||
              (svc.namespace ?? "").toLowerCase().includes(query),
          )
        : rc.services,
    }))
    .filter(
      (rc) =>
        !query || rc.clusterId.toLowerCase().includes(query) || rc.services.length > 0,
    );

  if (connectedClusters.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center text-slate-400 text-sm">
        No remote cluster endpoints discovered.
      </div>
    );
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden gap-2">
      <div className="flex items-center justify-between gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-500 pointer-events-none" />
          <Input
            placeholder="Filter by cluster or service..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-7 h-7 text-xs bg-slate-800 border-slate-700 text-slate-200 placeholder:text-slate-500 focus-visible:ring-slate-600"
          />
        </div>
        <Button
          size="sm"
          variant="ghost"
          className="cursor-pointer text-slate-400 hover:text-white gap-1"
          onClick={onRefresh}
        >
          <RefreshCw className="w-3 h-3" />
          Refresh
        </Button>
      </div>

      <div className="flex-1 overflow-auto flex flex-col gap-4">
        {filteredClusters.length === 0 ? (
          <div className="flex flex-1 items-center justify-center text-slate-400 text-sm">
            No results match your search.
          </div>
        ) : (
          filteredClusters.map((rc) => (
            <div key={rc.clusterId}>
              <div className="flex items-center gap-2 mb-1 px-1">
                <span className="text-sm font-semibold text-slate-200 font-mono">
                  {rc.clusterId}
                </span>
                <span className="text-xs text-slate-500">
                  {rc.services.length} service{rc.services.length !== 1 ? "s" : ""}
                </span>
              </div>

              <div className="rounded-lg border border-slate-700 overflow-hidden">
                <Table className="w-full text-xs">
                  <TableHeader className="bg-slate-800">
                    <TableRow className="border-slate-700 hover:bg-transparent">
                      <TableHead className="text-slate-400 font-semibold">
                        Service
                      </TableHead>
                      <TableHead className="text-slate-400 font-semibold">
                        Workload
                      </TableHead>
                      <TableHead className="text-slate-400 font-semibold">
                        Namespace
                      </TableHead>
                      <TableHead className="text-slate-400 font-semibold">
                        Addresses
                      </TableHead>
                      <TableHead className="text-slate-400 font-semibold">
                        Port
                      </TableHead>
                      <TableHead className="text-slate-400 font-semibold">
                        Health
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rc.services.map((svc, idx) => (
                      <TableRow
                        key={`${svc.serviceFqdn}:${svc.port}:${idx}`}
                        className="border-slate-800 hover:bg-slate-800/50"
                      >
                        <TableCell
                          className="font-mono text-white max-w-45 truncate"
                          title={svc.serviceFqdn}
                        >
                          {svc.serviceFqdn}
                        </TableCell>
                        <TableCell
                          className="text-slate-300 max-w-45 truncate"
                          title={svc.workloadName}
                        >
                          {svc.workloadName || "—"}
                        </TableCell>
                        <TableCell className="text-slate-300">
                          {svc.namespace || "—"}
                        </TableCell>
                        <TableCell className="font-mono text-slate-300 whitespace-nowrap">
                          {svc.addresses.join(", ") || "—"}
                        </TableCell>
                        <TableCell className="font-mono text-slate-300">
                          {svc.port}
                        </TableCell>
                        <TableCell>
                          <span
                            className={`inline-block w-2 h-2 rounded-full ${
                              svc.healthy ? "bg-green-400" : "bg-red-500"
                            }`}
                            title={svc.healthy ? "healthy" : "unhealthy"}
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};
