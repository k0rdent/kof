import { Pod } from "./Pod"
import { PodResponse, Target } from "./PrometheusTarget"

export interface NodeData {
    pods: Record<string, PodResponse>
}

export class Node {
    name: string
    private _pods: Record<string, Pod> = {}
    private _rawPods: Record<string, PodResponse> = {}

    constructor(name: string, pods: Record<string, PodResponse>) {
        this.name = name
        this._rawPods = pods
        for (const [podName, resp] of Object.entries(pods)) {
            this._pods[podName] = new Pod(podName, resp)
        }
    }

    public get pods(): Pod[] {
        return Object.values(this._pods)
    }

    public get rawPods(): Record<string, PodResponse> {
        return this._rawPods
    }

    public get targets(): Target[] {
        return Object.values(this._pods).flatMap(pod => pod.targets)
    }

    public get hasPods(): boolean {
        return this.pods.length > 0
    }

    public filterTargets(filterFn: (t: Target) => boolean): Node {
        const newPods: Record<string, PodResponse> = {}

        for (const pod of this.pods
            .map(p => p.filterTargets(filterFn))
            .filter(p => p.hasTargets)
        ) {
            newPods[pod.name] = pod.response
        }

        return new Node(this.name, newPods)
    }
}
