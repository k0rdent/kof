import { Pod } from "./Pod"
import { Target } from "./PrometheusTarget"


export interface NodeData {
    name: string
    pods: Pod[]
}

export class Node {
    name: string
    pods: Pod[] = []

    constructor(data: NodeData) {
        this.name = data.name
        data.pods.forEach(pod => this.pods.push(new Pod(pod.name, pod.response)))
    }

    public get targets(): Target[] {
        return this.pods.flatMap(pod => pod.response.data.activeTargets)
    }

    public get hasPods(): boolean {
        return this.pods.length > 0
    }

    public filterTargets(filterFn: (target: Target) => boolean): Node {
        return new Node({
            name: this.name,
            pods: this.pods
                .map(pod => pod.filterTargets(filterFn))
                .filter(pod => pod.hasTargets)
        })
    }
}
