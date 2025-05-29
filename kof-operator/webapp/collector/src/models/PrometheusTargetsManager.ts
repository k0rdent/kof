import { Cluster, ClustersData } from "./Cluster";
import { Target } from "./PrometheusTarget";


export class PrometheusTargetsManager {
    private _clusters: Record<string, Cluster> = {};

    constructor(data: ClustersData) {
        for (const [clusterName, cluster] of Object.entries(data.clusters)) {
            this._clusters[clusterName] = new Cluster(clusterName, cluster.nodes);
        }
    }

    public get clusters(): Cluster[] {
        return Object.values(this._clusters);
    }

    public get clustersCount(): number {
        return Object.keys(this._clusters).length;
    }

    public get targets(): Target[] {
        return Object.values(this._clusters).flatMap(cluster => cluster.targets);
    }

    public findCluster(name: string): Cluster | undefined {
        return this._clusters[name];
    }

    public filterClustersByNames(names: string[]): Record<string, Cluster> {
        return names.reduce<Record<string, Cluster>>((filtered, name) => {
            const cluster = this.findCluster(name);
            if (cluster) {
                filtered[name] = cluster;
            }
            return filtered;
        }, {});
    }
}
