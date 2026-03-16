import { JSX } from "react";
import { GraphNode } from "./types";

interface InfoTabProps {
  node: GraphNode;
}

export const InfoTab = ({ node }: InfoTabProps): JSX.Element => (
  <dl className="space-y-4 text-sm mt-4">
    <div>
      <dt className="text-slate-400 text-xs uppercase tracking-wider mb-1">
        Cluster name
      </dt>
      <dd className="text-white font-mono">{node.name}</dd>
    </div>
    <div>
      <dt className="text-slate-400 text-xs uppercase tracking-wider mb-1">
        Cluster Namespace
      </dt>
      <dd className="text-white font-mono">{node.namespace}</dd>
    </div>
    <div>
      <dt className="text-slate-400 text-xs uppercase tracking-wider mb-1">
        Cluster Role
      </dt>
      <dd className="text-white font-mono">{node.role}</dd>
    </div>
  </dl>
);
