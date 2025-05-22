import { Target, Cluster } from "@/models/PrometheusTarget";

export function getTargets(cluster: Cluster): Target[];
export function getTargets(clusters: Cluster[]): Target[];

export function getTargets(clusterOrClusters: Cluster | Cluster[]): Target[] {
  if (Array.isArray(clusterOrClusters)) {
    return clusterOrClusters.flatMap((cluster) => getTarget(cluster));
  } else {
    return getTarget(clusterOrClusters);
  }
}

function getTarget(cluster: Cluster): Target[] {
  return cluster.nodes
    .flatMap((node) => node.pods)
    .flatMap((pod) => pod.response.data.activeTargets) ?? [];
}