import { useClusterEndpoints } from "@/providers/istio/IstioClusterEndpointsProvider";
import { MeshLink, MeshNode } from "@/providers/istio/IstioMeshProvider";
import { JSX, useCallback, useEffect, useMemo, useRef, useState } from "react";
import ForceGraph2D, {
  ForceGraphMethods,
  LinkObject,
  NodeObject,
} from "react-force-graph-2d";
import { ClusterDetailSheet } from "./ClusterDetailSheet";
import { COLOR, drawNode, getNodeId, nodeKey } from "./constants";
import { Legend } from "./Legend";
import { LinkInfoPanel } from "./LinkInfoPanel";
import { GraphLink, GraphNode } from "./types";

interface MeshGraphViewProps {
  data: { nodes: MeshNode[]; links: MeshLink[] };
}

export const MeshGraphView = ({ data }: MeshGraphViewProps): JSX.Element => {
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<ForceGraphMethods | undefined>(undefined);

  const [dimensions, setDimensions] = useState({ width: 800, height: 600 });
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [selectedLink, setSelectedLink] = useState<GraphLink | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  const {
    data: endpointsData,
    loading,
    errors,
    fetchForCluster,
  } = useClusterEndpoints();

  // Track container size with ResizeObserver
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const observer = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect;
      if (width > 0 && height > 0) setDimensions({ width, height });
    });

    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Zoom-to-fit after data update
  useEffect(() => {
    const t = setTimeout(() => graphRef.current?.zoomToFit(800, 200), 500);
    return () => clearTimeout(t);
  }, [data]);

  // Memoize graphData so ForceGraph2D doesn't restart the simulation on
  // every re-render unrelated to the actual mesh data changing.
  const graphData = useMemo(() => {
    const rawLinks = data.links.map((l) => ({ ...l }));
    const pairSet = new Set(rawLinks.map((l) => `${l.source}|${l.target}`));
    const links = rawLinks.map((l) => ({
      ...l,
      curvature: pairSet.has(`${l.target}|${l.source}`) ? 0.3 : 0,
    }));
    return { nodes: data.nodes.map((n) => ({ ...n })), links };
  }, [data.links, data.nodes]);

  const handleNodeClick = useCallback(
    (node: NodeObject) => {
      setSelectedLink(null);
      const n = node as GraphNode;
      setSelectedNode((prev) => (prev?.id === n.id ? null : n));
      setSheetOpen(true);
      fetchForCluster(n.id, n.namespace);
    },
    [fetchForCluster],
  );

  const handleLinkClick = useCallback((link: LinkObject) => {
    setSelectedNode(null);
    setSheetOpen(false);
    setSelectedLink((prev) => {
      const next = link as GraphLink;
      const isSame =
        getNodeId(prev?.source) === getNodeId(next.source) &&
        getNodeId(prev?.target) === getNodeId(next.target) &&
        prev?.secretName === next.secretName;
      return isSame ? null : next;
    });
  }, []);

  const handleBackgroundClick = useCallback(() => {
    setSelectedNode(null);
    setSelectedLink(null);
    setSheetOpen(false);
  }, []);

  const handleSheetClose = useCallback(() => {
    setSheetOpen(false);
    setSelectedNode(null);
  }, []);

  const linkColor = useCallback(
    (link: LinkObject): string => {
      if (!selectedLink) return "rgba(148,163,184,0.35)";
      const l = link as GraphLink;
      return getNodeId(l.source) === getNodeId(selectedLink.source) &&
        getNodeId(l.target) === getNodeId(selectedLink.target) &&
        l.secretName === selectedLink.secretName
        ? COLOR.White
        : "rgba(148,163,184,0.35)";
    },
    [selectedLink],
  );

  const selectedKey = selectedNode ? nodeKey(selectedNode) : null;

  return (
    <>
      <div
        ref={containerRef}
        className="relative flex w-full min-h-0 flex-1 rounded-xl overflow-hidden bg-slate-950 border border-slate-800"
      >
        <ForceGraph2D
          ref={graphRef}
          width={dimensions.width}
          height={dimensions.height}
          graphData={graphData}
          nodeId="id"
          nodeLabel="name"
          nodeCanvasObject={drawNode}
          linkColor={linkColor}
          linkWidth={3}
          linkCurvature="curvature"
          linkDirectionalArrowLength={3}
          linkDirectionalArrowRelPos={1}
          onNodeClick={handleNodeClick}
          onLinkClick={handleLinkClick}
          onBackgroundClick={handleBackgroundClick}
          backgroundColor={COLOR.Black}
          cooldownTime={3000}
        />

        {selectedLink && (
          <LinkInfoPanel link={selectedLink} onClose={() => setSelectedLink(null)} />
        )}
        <Legend />
      </div>

      {/* Cluster detail sheet — slides in from the right */}
      <ClusterDetailSheet
        open={sheetOpen}
        node={selectedNode}
        connectivity={selectedKey ? (endpointsData.get(selectedKey) ?? null) : null}
        isLoading={selectedKey ? loading.has(selectedKey) : false}
        error={selectedKey ? (errors.get(selectedKey) ?? null) : null}
        onOpenChange={(open) => !open && handleSheetClose()}
        onRefresh={() =>
          selectedNode && fetchForCluster(selectedNode.id, selectedNode.namespace)
        }
      />
    </>
  );
};
