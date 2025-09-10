import { Response } from "../../src/providers/victoria_metrics/VictoriaMetricsProvider";

export const fakeVictoriaResponse: Response = {
  clusters: {
    "aws-ue2": {
      "vmselect-cluster-0": {
        condition_ready_healthy: [
          {
            value: "healthy",
          },
        ],
        go_cgo_calls_count: [
          {
            value: 27074437,
          },
        ],
        go_cpu_count: [
          {
            value: 10,
          },
        ],
        go_gc_cpu_seconds_total: [
          {
            value: 270.463318726,
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
          },
        ],
        vm_rows_deleted_total: [
          {
            value: 3912,
          },
        ],
      },
    },
    mothership: {
      "vmstorage-cluster-2": {
        vm_concurrent_select_current: [
          {
            value: 0,
          },
        ],
        vm_concurrent_select_limit_reached_total: [
          {
            value: 151224,
          },
        ],
        vm_concurrent_select_limit_timeout_total: [
          {
            value: 0,
          },
        ],
      },
      "vlstorage-test-1": {
        vm_concurrent_select_current: [
          {
            value: 0,
          },
        ],
        vm_concurrent_select_limit_reached_total: [
          {
            value: 12738,
          },
        ],
        vm_concurrent_select_limit_timeout_total: [
          {
            value: 0,
          },
        ],
      },
    },
  },
} as const;
