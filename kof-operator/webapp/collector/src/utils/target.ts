import { Target } from "@/models/PrometheusTarget";

export function getTargetCountByHealth(targets: Target[], health: string): number {
  return targets.filter((target) => target.health === health).length;
}

export function getDuplicatedScrapeUrls(
  targetsMap: Record<string, Target[]>,
): string[] {
  return Object.entries(targetsMap)
    .filter(([, targets]) => {
      const nodes = targets.map((t) => t.node);
      return new Set(nodes).size < nodes.length;
    })
    .map(([key]) => key);
}
