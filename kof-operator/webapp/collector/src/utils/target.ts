import { Target } from "@/models/PrometheusTarget";

export function getTargetCountByHealth(targets: Target[], health: string): number {
    return targets.filter(target => target.health === health).length
}