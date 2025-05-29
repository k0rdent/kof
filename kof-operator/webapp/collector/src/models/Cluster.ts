import { Target } from "@/models/PrometheusTarget"
import { Node, NodeData } from "@/models/Node"

export interface ClusterData {
    nodes: Record<string, NodeData>
}

export interface ClustersData {
    clusters: Record<string, ClusterData>
}

export class Cluster {
    name: string
    private _nodes: Record<string, Node> = {}
    private _nodesData: Record<string, NodeData> = {}

    constructor(name: string, nodes: Record<string, NodeData>) {
        this.name = name
        this._nodesData = nodes
        for (const [nodeName, resp] of Object.entries(nodes)) {
            this._nodes[nodeName] = new Node(nodeName, resp.pods)
        }
    }

    public get nodes(): Node[] {
        return Object.values(this._nodes)
    }

    public get rawNodes(): Record<string, NodeData> {
        return this._nodesData
    }

    public get targets(): Target[] {
        return Object.values(this._nodes).flatMap(node => node.targets)
    }

    public get hasNodes(): boolean {
        return this.nodes.length > 0
    }

    public findNode(name: string): Node | undefined {
        return this._nodes[name]
    }

    public findNodes(names: string[]): Record<string, NodeData> {
        return names.reduce<Record<string, NodeData>>((filtered, name) => {
            const nodeData = this._nodesData[name]
            if (nodeData) {
                filtered[name] = nodeData
            }
            return filtered
        }, {})
    }

    public filterTargets(filterFn: (t: Target) => boolean): Cluster {
        const newNodesData: Record<string, NodeData> = {}

        for (const node of this.nodes
            .map(n => n.filterTargets(filterFn))
            .filter(n => n.hasPods)
        ) {
            newNodesData[node.name] = { pods: node.rawPods }
        }

        return new Cluster(this.name, newNodesData)
    }
}
