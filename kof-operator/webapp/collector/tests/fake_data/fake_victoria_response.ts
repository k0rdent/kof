import { ClusterData } from "../../src/components/pages/collectorPage/models";

type Response = {
  clusters: Record<string, ClusterData>;
};

export const fakeVictoriaResponse: Response = {
  clusters: {
    "aws-ue2": {
      customResource: {
        victoriaMetrics: {
          pods: {
            "vmselect-cluster-0": {
              metrics: {
                condition_ready_healthy: [
                  {
                    value: "healthy",
                    labels: undefined,
                  },
                ],
                go_cgo_calls_count: [
                  {
                    value: 27074437,
                    labels: undefined,
                  },
                ],
                go_cpu_count: [
                  {
                    value: 10,
                    labels: undefined,
                  },
                ],
                go_gc_cpu_seconds_total: [
                  {
                    value: 270.463318726,
                    labels: undefined,
                  },
                ],
                go_gc_duration_seconds: [
                  {
                    labels: {
                      quantile: "0",
                    },
                    value: 0.000010207,
                  },
                  {
                    labels: {
                      quantile: "0.25",
                    },
                    value: 0.000025375,
                  },
                  {
                    labels: {
                      quantile: "0.5",
                    },
                    value: 0.000032166,
                  },
                  {
                    labels: {
                      quantile: "0.75",
                    },
                    value: 0.000048042,
                  },
                  {
                    labels: {
                      quantile: "1",
                    },
                    value: 0.001657666,
                  },
                ],
                go_gc_duration_seconds_count: [
                  {
                    value: 194227,
                    labels: undefined,
                  },
                ],
                vm_rows_deleted_total: [
                  {
                    value: 3912,
                    labels: undefined,
                  },
                ],
              },
            },
          },
        },
      },
    },
    mothership: {
      customResource: {
        victoriaLogs: {
          pods: {
            "vmstorage-cluster-2": {
              metrics: {
                vm_concurrent_select_current: [
                  {
                    value: 0,
                    labels: undefined,
                  },
                ],
                vm_concurrent_select_limit_reached_total: [
                  {
                    value: 151224,
                    labels: undefined,
                  },
                ],
                vm_concurrent_select_limit_timeout_total: [
                  {
                    value: 0,
                    labels: undefined,
                  },
                ],
              },
            },
            "vlstorage-test-1": {
              metrics: {
                vm_concurrent_select_current: [
                  {
                    value: 0,
                    labels: undefined,
                  },
                ],
                vm_concurrent_select_limit_reached_total: [
                  {
                    value: 12738,
                    labels: undefined,
                  },
                ],
                vm_concurrent_select_limit_timeout_total: [
                  {
                    value: 0,
                    labels: undefined,
                  },
                ],
              },
            },
          },
        },
      },
    },
  },
} as const;
