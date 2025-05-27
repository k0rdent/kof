import { Cluster, ClustersData } from "./Cluster";
import { Target } from "./PrometheusTarget";


export class PrometheusTargetsManager {
    private _clusters: Cluster[] = []

    constructor(data: ClustersData) {
        data.clusters.forEach(cluster => this._clusters.push(new Cluster(cluster)))
    }

    public get clusters(): Cluster[] {
        return this._clusters
    }

    public get clustersCount(): number {
        return this.clusters.length
    }

    public get targets(): Target[] {
        return this.clusters.flatMap(cluster => cluster.targets)
    }

    public findCluster(name: string): Cluster | undefined {
        return this.clusters.find(cluster => cluster.name === name)
    }

    public filterClustersByNames(names: string[]): Cluster[] {
        return this.clusters.filter(cluster => names.includes(cluster.name)) ?? []
    }
}