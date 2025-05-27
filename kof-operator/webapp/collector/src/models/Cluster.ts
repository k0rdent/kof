import { Target } from "@/models/PrometheusTarget"
import { Node } from "@/models/Node"

export interface ClusterData {
    name: string
    nodes: Node[]
}

export interface ClustersData {
    clusters: ClusterData[]
}

export class Cluster {
    name: string
    nodes: Node[] = []

    constructor(data: ClusterData) {
        this.name = data.name
        data.nodes.forEach(node => this.nodes.push(new Node(node)))
    }

    public get targets(): Target[] {
        return this.nodes.flatMap(node => node.targets)
    }

    public get hasNodes(): boolean {
        return this.nodes.length > 0
    }

    public findNode(name: string): Node | undefined {
        return this.nodes.find(node => node.name === name)
    }

    public findNodes(names: string[]): Node[] {
        return this.nodes.filter(node => names.includes(node.name))
    }

    public filterTargets(filterFn: (target: Target) => boolean): Cluster {
        return new Cluster({
            name: this.name,
            nodes: this.nodes
                .map(node => node.filterTargets(filterFn))
                .filter(node => node.hasPods)
        })
    }
}
