import { PodResponse, Target } from "./PrometheusTarget"

export class Pod {
    constructor(
        public readonly name: string,
        public readonly response: PodResponse
    ) { }

    public get targets(): Target[] {
        return this.response.data.activeTargets
    }

    public get hasTargets(): boolean {
        return this.targets.length > 0
    }

    public filterTargets(filterFn: (target: Target) => boolean): Pod {
        const filteredTargets = this.targets.filter(filterFn)

        if (filteredTargets.length === this.targets.length) {
            return this
        }

        return new Pod(this.name, {
            ...this.response,
            data: {
                ...this.response.data,
                activeTargets: filteredTargets
            }
        })
    }
}