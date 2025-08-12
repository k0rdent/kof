const VM_TYPES = [
  "vlselect",
  "vlinsert",
  "vlstorage",
  "vmselect",
  "vminsert",
  "vmstorage",
  "vm",
] as const;

export type VmTypes = (typeof VM_TYPES)[number];

const NAME_TYPE_MAPPING: Record<VmTypes, string> = {
  vlselect: "VictoriaLogs Select",
  vlinsert: "VictoriaLogs Insert",
  vlstorage: "VictoriaLogs Storage",
  vmselect: "VictoriaMetrics Select",
  vminsert: "VictoriaMetrics Insert",
  vmstorage: "VictoriaMetrics Storage",
  vm: "VictoriaMetrics",
};

export function getVictoriaNameType(name: string): string {
  const normalized = (name ?? "").trim().toLowerCase();
  const match = VM_TYPES.find(k => normalized.includes(k));
  return match ? NAME_TYPE_MAPPING[match] : NAME_TYPE_MAPPING.vm;
}

export function getVictoriaType(name: string): VmTypes {
  const normalized = (name ?? "").trim().toLowerCase();
  const match = VM_TYPES.find(k => normalized.includes(k));
  return match ?? "vm";
}

