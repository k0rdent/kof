import { Badge } from "@/components/generated/ui/badge";
import { Button } from "@/components/generated/ui/button";
import { Separator } from "@/components/generated/ui/separator";
import Loader from "@/components/shared/Loader";
import { MeshLink, MeshNode, useIstioMesh } from "@/providers/istio/IstioMeshProvider";
import { X } from "lucide-react";
import { JSX, useCallback, useEffect, useMemo, useRef, useState } from "react";
import ForceGraph2D, {
  ForceGraphMethods,
  LinkObject,
  NodeObject,
} from "react-force-graph-2d";

interface GraphNode extends NodeObject {
  id: string;
  name: string;
  role: string;
}

interface GraphLink extends LinkObject {
  secretName: string;
}

// Colour helpers
const COLOR: Record<string, string> = {
  Purple: "#6366f1",
  Orange: "#f59e0b",
  Green: "#10b981",
  Gray: "#94a3b8",
  White: "#FFFFFF",
  Black: "#020617",
};

const ROLE_COLORS: Record<string, string> = {
  management: COLOR.Purple,
  regional: COLOR.Orange,
  child: COLOR.Green,
};

const roleColor = (role: string): string => ROLE_COLORS[role] ?? COLOR.Gray;

const NODE_RADIUS = 5;
const LABEL_OFFSET = NODE_RADIUS + 5;

const getNodeId = (node: string | number | NodeObject | undefined): string =>
  typeof node === "string"
    ? node
    : typeof node === "number"
      ? String(node)
      : node?.id !== undefined
        ? String(node.id)
        : "";

function drawNode(
  node: NodeObject,
  ctx: CanvasRenderingContext2D,
  globalScale: number,
) {
  const n = node as GraphNode;
  const x = n.x ?? 0;
  const y = n.y ?? 0;
  const fontSize = Math.max(6 / globalScale, 3);

  ctx.beginPath();
  ctx.arc(x, y, NODE_RADIUS, 0, 2 * Math.PI);
  ctx.fillStyle = roleColor(n.role);
  ctx.fill();

  ctx.font = `${fontSize}px sans-serif`;
  ctx.fillStyle = COLOR.White;
  ctx.textAlign = "center";
  ctx.fillText(n.name, x, y + LABEL_OFFSET);
}

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

interface MeshBodyProps {
  isLoading: boolean;
  data: { nodes: MeshNode[]; links: MeshLink[] } | null;
  error: Error | null;
  onRetry: () => void;
}

const MeshBody = ({ isLoading, data, error, onRetry }: MeshBodyProps): JSX.Element => {
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

interface MeshGraphViewProps {
  data: { nodes: MeshNode[]; links: MeshLink[] };
}

const MeshGraphView = ({ data }: MeshGraphViewProps): JSX.Element => {
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<ForceGraphMethods | undefined>(undefined);

  const [dimensions, setDimensions] = useState({ width: 800, height: 600 });
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [selectedLink, setSelectedLink] = useState<GraphLink | null>(null);

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

  const handleNodeClick = useCallback((node: NodeObject) => {
    setSelectedLink(null);
    setSelectedNode((prev) => {
      const n = node as GraphNode;
      return prev?.id === n.id ? null : n;
    });
  }, []);

  const handleLinkClick = useCallback((link: LinkObject) => {
    setSelectedNode(null);
    setSelectedLink((prev) => {
      const next = link as GraphLink;

      const nextSrc = getNodeId(next.source);
      const nextTgt = getNodeId(next.target);

      const prevSrc = getNodeId(prev?.source);
      const prevTgt = getNodeId(prev?.target);

      const isSame =
        prevSrc === nextSrc &&
        prevTgt === nextTgt &&
        prev?.secretName === next.secretName;

      return isSame ? null : next;
    });
  }, []);

  const handleBackgroundClick = useCallback(() => {
    setSelectedNode(null);
    setSelectedLink(null);
  }, []);

  const linkColor = useCallback(
    (link: LinkObject): string => {
      if (!selectedLink) return "rgba(148,163,184,0.35)";
      const l = link as GraphLink;
      const srcId = getNodeId(l.source);
      const tgtId = getNodeId(l.target);
      const selSrc = getNodeId(selectedLink.source);
      const selTgt = getNodeId(selectedLink.target);
      return srcId === selSrc &&
        tgtId === selTgt &&
        l.secretName === selectedLink.secretName
        ? COLOR.White
        : "rgba(148,163,184,0.35)";
    },
    [selectedLink],
  );

  const clearSelection = useCallback(() => {
    setSelectedNode(null);
    setSelectedLink(null);
  }, []);

  return (
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

      {(selectedNode || selectedLink) && (
        <InfoPanel
          node={selectedNode}
          link={selectedLink}
          onClose={clearSelection}
        />
      )}
      <Legend />
    </div>
  );
};

interface InfoPanelProps {
  node?: GraphNode | null;
  link?: GraphLink | null;
  onClose: () => void;
}

const InfoPanel = ({ node, link, onClose }: InfoPanelProps): JSX.Element => {
  return (
    <div className="absolute top-4 right-4 w-64 rounded-xl border border-white/10 bg-slate-900/90 p-4 shadow-2xl backdrop-blur-sm z-10">
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm font-semibold text-white">
          {node ? "Cluster" : "Connection"}
        </span>
        <X
          onClick={onClose}
          className="w-4 cursor-pointer text-slate-400 hover:text-white transition-colors"
        />
      </div>

      {node && (
        <dl className="space-y-1 text-xs">
          <dt className="text-slate-400">Name</dt>
          <dd className="text-white font-mono">{node.name}</dd>
          <dt className="text-slate-400 mt-2">Role</dt>
          <dd>
            <Badge
              className="text-[10px] font-semibold"
              style={{ backgroundColor: roleColor(node.role) }}
            >
              {node.role}
            </Badge>
          </dd>
        </dl>
      )}

      {link && (
        <dl className="space-y-2 text-xs">
          <div>
            <dt className="text-slate-400">From</dt>
            <dd className="text-white font-mono break-all">{getNodeId(link.source)}</dd>
          </div>
          <div>
            <dt className="text-slate-400">To</dt>
            <dd className="text-white font-mono break-all">{getNodeId(link.target)}</dd>
          </div>
          <div>
            <dt className="text-slate-400">Secret</dt>
            <dd className="text-white font-mono break-all">{link.secretName || "—"}</dd>
          </div>
        </dl>
      )}
    </div>
  );
};

const Legend = (): JSX.Element => (
  <div className="absolute bottom-4 left-4 rounded-xl border border-white/10 bg-slate-900/90 p-3 backdrop-blur-sm z-10">
    <p className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-2">
      Cluster role
    </p>
    {Object.entries(ROLE_COLORS).map(([role, color]) => (
      <div key={role} className="flex items-center gap-2 mb-1">
        <span
          className="inline-block w-3 h-3 rounded-full"
          style={{ backgroundColor: color }}
        />
        <span className="text-xs text-slate-200 capitalize">{role}</span>
      </div>
    ))}
  </div>
);

export default IstioMeshesPage;
