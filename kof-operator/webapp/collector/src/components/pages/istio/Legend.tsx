import { Separator } from "@/components/generated/ui/separator";
import { JSX } from "react";
import { ROLE_COLORS } from "./constants";

export const Legend = (): JSX.Element => (
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
    <Separator className="my-2 bg-slate-700" />
    <p className="text-[10px] text-slate-400">Click a node to see endpoints</p>
  </div>
);
