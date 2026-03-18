import { X } from "lucide-react";
import { JSX } from "react";
import { getNodeId } from "./constants";
import { GraphLink } from "./types";

interface LinkInfoPanelProps {
  link: GraphLink;
  onClose: () => void;
}

export const LinkInfoPanel = ({ link, onClose }: LinkInfoPanelProps): JSX.Element => (
  <div className="absolute top-4 right-4 w-64 rounded-xl border border-white/10 bg-slate-900/90 p-4 shadow-2xl backdrop-blur-sm z-10">
    <div className="flex items-center justify-between mb-3">
      <span className="text-sm font-semibold text-white">Connection</span>
      <X
        onClick={onClose}
        className="w-4 cursor-pointer text-slate-400 hover:text-white transition-colors"
      />
    </div>
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
  </div>
);
