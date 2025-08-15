export const METRICS = {
  OTELCOL_EXPORTER_PROM_WRITE_CONSUMERS: {
    name: "otelcol_exporter_prometheusremotewrite_consumers",
    hint: "Number of configured workers to use to fan out the outgoing requests",
  },
  OTELCOL_EXPORTER_PROM_WRITE_SENT_BATCHES: {
    name: "otelcol_exporter_prometheusremotewrite_sent_batches_total",
    hint: "Number of remote write request batches sent to the remote write endpoint regardless of success or failure",
  },
  OTELCOL_EXPORTER_PROM_WRITE_TRANS_RATIO: {
    name: "otelcol_exporter_prometheusremotewrite_translated_time_series_ratio_total",
    hint: "Number of Prometheus time series that were translated from OTel metrics",
  },
  OTELCOL_EXPORTER_QUEUE_CAPACITY: {
    name: "otelcol_exporter_queue_capacity",
    hint: "Fixed capacity of the sending queue, in batches",
  },
  OTELCOL_EXPORTER_QUEUE_SIZE: {
    name: "otelcol_exporter_queue_size",
    hint: "Current size of the sending queue, in batches",
  },
  OTELCOL_EXPORTER_SEND_FAILED_LOG_RECORDS: {
    name: "otelcol_exporter_send_failed_log_records_total",
    hint: "Total log records the exporter failed to send",
  },
  OTELCOL_EXPORTER_SEND_FAILED_METRIC_POINTS: {
    name: "otelcol_exporter_send_failed_metric_points_total",
    hint: "Total metric points the exporter failed to send",
  },
  OTELCOL_EXPORTER_SENT_LOG_RECORDS: {
    name: "otelcol_exporter_sent_log_records_total",
    hint: "Number of logs successfully sent to destination",
  },
  OTELCOL_EXPORTER_SENT_METRIC_POINTS: {
    name: "otelcol_exporter_sent_metric_points_total",
    hint: "Number of metric points successfully sent to destination",
  },
  OTELCOL_FILECONSUMER_OPEN_FILES: {
    name: "otelcol_fileconsumer_open_files_ratio",
    hint: "Number of open files",
  },
  OTELCOL_FILECONSUMER_READING_FILES: {
    name: "otelcol_fileconsumer_reading_files_ratio",
    hint: "Number of open files that are being read",
  },
  OTELCOL_PROCESS_MEMORY_RSS: {
    name: "otelcol_process_memory_rss_bytes",
    hint: "Total physical memory (resident set size)",
  },
  OTELCOL_PROCESS_RUNTIME_HEAP_ALLOC: {
    name: "otelcol_process_runtime_heap_alloc_bytes",
    hint: "Size of allocated heap objects",
  },
  OTELCOL_PROCESS_RUNTIME_TOTAL_SYS_MEMORY: {
    name: "otelcol_process_runtime_total_sys_memory_bytes",
    hint: "Total size of memory obtained from the OS",
  },
  OTELCOL_PROCESS_UPTIME_SECONDS: {
    name: "otelcol_process_uptime_seconds_total",
    hint: "Uptime of the process",
  },
  OTELCOL_PROCESSOR_BATCH_SEND_SIZE: {
    name: "otelcol_processor_batch_batch_send_size",
    hint: "Number of units in the batch",
  },
  OTELCOL_PROCESSOR_BATCH_SIZE_TRIGGER_SEND: {
    name: "otelcol_processor_batch_batch_size_trigger_send_total",
    hint: "Number of times the batch was sent due to a size trigger",
  },
  OTELCOL_PROCESSOR_BATCH_METADATA_CARDINALITY: {
    name: "otelcol_processor_batch_metadata_cardinality",
    hint: "Number of distinct metadata value combinations being processed",
  },
  OTELCOL_PROCESSOR_BATCH_TIMEOUT_TRIGGER_SEND: {
    name: "otelcol_processor_batch_timeout_trigger_send_total",
    hint: "Number of times the batch was sent due to a timeout trigger",
  },
  OTELCOL_PROCESSOR_INCOMING_ITEMS: {
    name: "otelcol_processor_incoming_items_total",
    hint: "Number of items passed to the processor",
  },
  OTELCOL_PROCESSOR_OUTGOING_ITEMS: {
    name: "otelcol_processor_outgoing_items_total",
    hint: "Number of items emitted from the processor.",
  },
  OTELCOL_RECEIVER_ACCEPTED_LOG_RECORDS: {
    name: "otelcol_receiver_accepted_log_records_total",
    hint: "Number of log records successfully pushed into the pipeline",
  },
  OTELCOL_RECEIVER_ACCEPTED_METRIC_POINTS: {
    name: "otelcol_receiver_accepted_metric_points_total",
    hint: "Number of metric points successfully pushed into the pipeline",
  },
  OTELCOL_RECEIVER_REFUSED_LOG_RECORDS: {
    name: "otelcol_receiver_refused_log_records_total",
    hint: "Number of log records that could not be pushed into the pipeline",
  },
  OTELCOL_RECEIVER_REFUSED_METRIC_POINTS: {
    name: "otelcol_receiver_refused_metric_points_total",
    hint: "Number of metric points that could not be pushed into the pipeline",
  },
  CONDITION_READY_HEALTHY: {
    name: "condition_ready_healthy",
    hint: "Container condition",
  },
  CONDITION_READY_MESSAGE: {
    name: "condition_ready_message",
    hint: "Container condition message",
  },
  CONDITION_READY_REASON: {
    name: "condition_ready_reason",
    hint: "Container condition reason",
  },
  CONTAINER_RESOURCE_CPU_LIMIT: {
    name: "container_resource_cpu_limit",
    hint: "CPU limit for the container",
  },
  CONTAINER_RESOURCE_CPU_USAGE: {
    name: "container_resource_cpu_usage",
    hint: "Current CPU usage for the container",
  },
  CONTAINER_RESOURCE_MEMORY_LIMIT: {
    name: "container_resource_memory_limit",
    hint: "Memory limit for the container",
  },
  CONTAINER_RESOURCE_MEMORY_USAGE: {
    name: "container_resource_memory_usage",
    hint: "Current memory usage of the container",
  },
} as const;

export const VICTORIA_METRICS = {
  VM_HTTP_REQUESTS_ALL_TOTAL: {
    name: "vm_http_requests_all_total",
    hint: "Total HTTP requests received/handled by VictoriaMetrics",
  },
  VM_HTTP_REQUEST_DURATION_SECONDS_SUM: {
    name: "vm_http_request_duration_seconds_sum",
    hint: "Cumulative sum of HTTP request durations in seconds",
  },
  VM_HTTP_REQUEST_DURATION_SECONDS_COUNT: {
    name: "vm_http_request_duration_seconds_count",
    hint: "Total number of HTTP requests observed",
  },
  VM_HTTP_REQUEST_ERRORS_TOTAL: {
    name: "vm_http_request_errors_total",
    hint: "Total HTTP request errors (failed/errored requests)",
  },
  VM_HTTP_CONN_TIMEOUT_CLOSED_CONNS_TOTAL: {
    name: "vm_http_conn_timeout_closed_conns_total",
    hint: "Total HTTP connections closed due to timeout",
  },
  VM_TCPLISTENER_CONNS: {
    name: "vm_tcplistener_conns",
    hint: "Current active TCP connections on the listener",
  },
  VM_TCPLISTENER_READ_BYTES_TOTAL: {
    name: "vm_tcplistener_read_bytes_total",
    hint: "Total bytes read by the TCP listener",
  },
  VM_TCPLISTENER_WRITTEN_BYTES_TOTAL: {
    name: "vm_tcplistener_written_bytes_total",
    hint: "Total bytes written by the TCP listener",
  },
  VM_TCPLISTENER_ERRORS_TOTAL: {
    name: "vm_tcplistener_errors_total",
    hint: "Total errors encountered by the TCP listener",
  },
  VM_TCPLISTENER_ACCEPTS_TOTAL: {
    name: "vm_tcplistener_accepts_total",
    hint: "Total TCP connection accept events handled by the listener",
  },
  VM_AVAILABLE_CPU_CORES: {
    name: "vm_available_cpu_cores",
    hint: "Number of CPU cores available to the VictoriaMetrics instance",
  },
  VL_ERRORS_TOTAL: {
    name: "vl_errors_total",
    hint: "Total number of errors recorded by VictoriaLogs",
  },
  VL_HTTP_ERRORS_TOTAL: {
    name: "vl_http_errors_total",
    hint: "Total number of HTTP errors recorded by VictoriaLogs",
  },
  VL_UDP_ERRORS_TOTAL: {
    name: "vl_udp_errors_total",
    hint: "Total number of UDP errors recorded by VictoriaLogs",
  },
  VL_UDP_REQESTS_TOTAL: {
    name: "vl_udp_reqests_total",
    hint: "Total number of UDP requests received by VictoriaLogs",
  },

  VL_DATA_SIZE_BYTES: {
    name: "vl_data_size_bytes",
    hint: "Total size of stored data",
  },
  VL_FREE_DISK_SPACE_BYTES: {
    name: "vl_free_disk_space_bytes",
    hint: "Available free disk space for VictoriaLogs",
  },
  VL_PARTITIONS: {
    name: "vl_partitions",
    hint: "Number of storage partitions",
  },
  VL_STORAGE_IS_READ_ONLY: {
    name: "vl_storage_is_read_only",
    hint: "Flag indicating storage is read-only",
  },
  VL_ROWS_INGESTED_TOTAL: {
    name: "vl_rows_ingested_total",
    hint: "Total rows successfully ingested by VictoriaLogs",
  },
  VL_BYTES_INGESTED_TOTAL: {
    name: "vl_bytes_ingested_total",
    hint: "Total bytes successfully ingested by VictoriaLogs",
  },
  VL_ROWS_DROPPED_TOTAL: {
    name: "vl_rows_dropped_total",
    hint: "Total rows dropped by VictoriaLogs",
  },
  VL_TOO_LONG_LINES_SKIPPED_TOTAL: {
    name: "vl_too_long_lines_skipped_total",
    hint: "Total lines skipped because they exceeded the maximum length",
  },
  VL_STORAGE_PARTS: {
    name: "vl_storage_parts",
    hint: "Total number of storage parts/segments",
  },
  VL_STORAGE_ROWS: {
    name: "vl_storage_rows",
    hint: "Total number of rows stored",
  },
  VL_STORAGE_BLOCKS: {
    name: "vl_storage_blocks",
    hint: "Total number of storage blocks",
  },
  PROCESS_IO_READ_BYTES_TOTAL: {
    name: "process_io_read_bytes_total",
    hint: "Total bytes read by the process",
  },
  PROCESS_IO_WRITTEN_BYTES_TOTAL: {
    name: "process_io_written_bytes_total",
    hint: "Total bytes written by the process",
  },
  PROCESS_IO_READ_SYSCALLS_TOTAL: {
    name: "process_io_read_syscalls_total",
    hint: "Total read syscalls made by the process",
  },
  PROCESS_IO_WRITE_SYSCALLS_TOTAL: {
    name: "process_io_write_syscalls_total",
    hint: "Total write syscalls made by the process",
  },
  PROCESS_CPU_SECONDS_TOTAL: {
    name: "process_cpu_seconds_total",
    hint: "CPU time consumed by the process",
  },
  PROCESS_CPU_SECONDS_USER_TOTAL: {
    name: "process_cpu_seconds_user_total",
    hint: "CPU time spent in user mode",
  },
  PROCESS_CPU_SECONDS_SYSTEM_TOTAL: {
    name: "process_cpu_seconds_system_total",
    hint: "CPU time spent in kernel/system mode",
  },
  PROCESS_RESIDENT_MEMORY_BYTES: {
    name: "process_resident_memory_bytes",
    hint: "Current resident (RSS) memory size",
  },
  PROCESS_RESIDENT_MEMORY_PEAK_BYTES: {
    name: "process_resident_memory_peak_bytes",
    hint: "Peak resident (RSS) memory size",
  },
  GO_GOROUTINES: {
    name: "go_goroutines",
    hint: "Number of currently live goroutines",
  },
  GO_CGO_CALLS_COUNT: {
    name: "go_cgo_calls_count",
    hint: "Total cgo calls made by the Go program",
  },
  GO_GOMAXPROCES: {
    name: "go_gomaxprocs",
    hint: "Max OS threads usable by Go program",
  },
  GO_GC_CPU_SECONDS_TOTAL: {
    name: "go_gc_cpu_seconds_total",
    hint: "Total CPU time spent in garbage collection",
  },
  GO_MEMSTATS_HEAP_ALLOC_BYTES: {
    name: "go_memstats_heap_alloc_bytes",
    hint: "Current heap memory allocated by the Go runtime",
  },
  GO_MEMSTATS_HEAP_INUSE_BYTES: {
    name: "go_memstats_heap_inuse_bytes",
    hint: "Current heap memory in use by the Go runtime",
  },
  GO_MEMSTATS_HEAP_IDLE_BYTES: {
    name: "go_memstats_heap_idle_bytes",
    hint: "Current heap memory that is idle and not in use",
  },
  GO_MEMSTATS_HEAP_OBJECTS: {
    name: "go_memstats_heap_objects",
    hint: "Number of allocated heap objects",
  },

  // VLSelect
  VLSELECT_BACKEND_CONNS: {
    name: "vlselect_backend_conns",
    hint: "Current active backend connections used by VLSelect",
  },
  VLSELECT_BACKEND_CONN_BYTES_READ_TOTAL: {
    name: "vlselect_backend_conn_bytes_read_total",
    hint: "Total bytes read from backend connections by VLSelect",
  },
  VLSELECT_BACKEND_CONN_BYTES_WRITTEN_TOTAL: {
    name: "vlselect_backend_conn_bytes_written_total",
    hint: "Total bytes written to backend connections by VLSelect",
  },
  VLSELECT_BACKEND_CONN_READ_ERRORS_TOTAL: {
    name: "vlselect_backend_conn_read_errors_total",
    hint: "Total read errors on backend connections used by VLSelect",
  },
  VLSELECT_BACKEND_CONN_WRITE_ERRORS_TOTAL: {
    name: "vlselect_backend_conn_write_errors_total",
    hint: "Total write errors on backend connections used by VLSelect",
  },
  VLSELECT_BACKEND_DIAL_ERRORS_TOTAL: {
    name: "vlselect_backend_dial_errors_total",
    hint: "Total backend dial errors encountered by VLSelect",
  },

  // VLInsert
  VLINSERT_BACKEND_CONNS: {
    name: "vlinsert_backend_conns",
    hint: "Current active backend connections used by VLInsert",
  },
  VLINSERT_BACKEND_CONNS_BYTE_WRITTEN: {
    name: "vlinsert_backend_conn_bytes_written_total",
    hint: "Total bytes written to backend connections by VLInsert",
  },
  VLINSERT_BACKEND_CONNS_BYTE_READ: {
    name: "vlinsert_backend_conn_bytes_read_total",
    hint: "Total bytes read from backend connections by VLInsert",
  },
  VLINSERT_BACKEND_CONNS_WRITES_TOTAL: {
    name: "vlinsert_backend_conn_writes_total",
    hint: "Total write operations to backend connections by VLInsert",
  },
  VLINSERT_BACKEND_CONNS_READS_TOTAL: {
    name: "vlinsert_backend_conn_reads_total",
    hint: "Total read operations from backend connections by VLInsert",
  },
  VLINSERT_BACKEND_CONNS_READ_ERRORS_TOTAL: {
    name: "vlinsert_backend_conn_read_errors_total",
    hint: "Total read errors on backend connections used by VLInsert",
  },
  VLINSERT_BACKEND_CONNS_WRITE_ERRORS_TOTAL: {
    name: "vlinsert_backend_conn_write_errors_total",
    hint: "Total write errors on backend connections used by VLInsert",
  },
  VLINSERT_BACKEND_DIALS_TOTAL: {
    name: "vlinsert_backend_dials_total",
    hint: "Total backend connection attempts made by VLInsert",
  },
  VLINSERT_BACKEND_DIALS_ERRORS_TOTAL: {
    name: "vlinsert_backend_dial_errors_total",
    hint: "Total backend failed connection attempts by VLInsert",
  },

  VM_RPC_ROWS_SENT_TOTAL: {
    name: "vm_rpc_rows_sent_total",
    hint: "Total rows sent to VMInsert",
  },
  VM_RPC_ROWS_PENDING: {
    name: "vm_rpc_rows_pending",
    hint: "Current number of rows pending to be sent via RPC",
  },
  VM_RPC_ROWS_PUSHED_TOTAL: {
    name: "vm_rpc_rows_pushed_total",
    hint: "Total rows successfully pushed to remote RPC peers",
  },
  VM_ROWS_INVALID_TOTAL: {
    name: "vm_rows_invalid_total",
    hint: "Total rows that were invalid and not inserted into VictoriaMetrics",
  },

  VM_RPC_VMSTORAGE_IS_REACHABLE: {
    name: "vm_rpc_vmstorage_is_reachable",
    hint: "Flag indicating if VMStorage is reachable",
  },
  VM_RPC_VMSTORAGE_IS_READ_ONLY: {
    name: "vm_rpc_vmstorage_is_read_only",
    hint: "Flag showing if VMStorage is read-only",
  },
  VM_RPC_DIAL_ERRORS_TOTAL: {
    name: "vm_rpc_dial_errors_total",
    hint: "Total RPC connection errors when contacting VMStorage",
  },
  VM_RPC_ROWS_DROPPED_ON_OVERLOAD_TOTAL: {
    name: "vm_rpc_rows_dropped_on_overload_total",
    hint: "Total rows dropped due to overload during RPC handling",
  },
  VM_RPC_ROWS_INCOMPLETELY_REPLICATED_TOTAL: {
    name: "vm_rpc_rows_incompletely_replicated_total",
    hint: "Total rows that were only partially replicated",
  },
  VM_RPC_REROUTES_TOTAL: {
    name: "vm_rpc_reroutes_total",
    hint: "Total RPC requests that were rerouted",
  },

  // VMSelect
  VM_TENANT_SELECT_REQUEST_TOTAL: {
    name: "vm_tenant_select_requests_total",
    hint: "Total tenant select requests received",
  },
  VM_METRIC_ROWS_READ_TOTAL: {
    name: "vm_metric_rows_read_total",
    hint: "Total metric rows read from storage",
  },
  VM_METRICS_ROWS_SKIPPED_TOTAL: {
    name: "vm_metric_rows_skipped_total",
    hint: "Total metric rows skipped during read operations",
  },
  VM_TMP_BLOCK_FILES_CREATED_TOTAL: {
    name: "vm_tmp_blocks_files_created_total",
    hint: "Total temporary block files created.",
  },
  VM_TMP_BLOCK_FILES_DIRECTORY_FREE_BYTES: {
    name: "vm_tmp_blocks_files_directory_free_bytes",
    hint: "Free space available in the temp block files directory",
  },
  VM_TMP_BLOCK_MAX_INMEMORY_FILE_SIZE_BYTES: {
    name: "vm_tmp_blocks_max_inmemory_file_size_bytes",
    hint: "Maximum size of in-memory temporary block file",
  },
  VM_ROLLUP_RESULT_CACHE_FULL_HITS_TOTAL: {
    name: "vm_rollup_result_cache_full_hits_total",
    hint: "Total full cache hits for rollup query results (exact match)",
  },
  VM_ROLLUP_RESULT_CACHE_PARTIAL_HITS_TOTAL: {
    name: "vm_rollup_result_cache_partial_hits_total",
    hint: "Total partial cache hits where rollup results were partially reused",
  },
  VM_ROLLUP_RESULT_CACHE_MISS_TOTAL: {
    name: "vm_rollup_result_cache_miss_total",
    hint: "Total rollup queries that missed the cache",
  },

  // VMStorage
  VM_ROWS_RECEIVED_BY_STORAGE_TOTAL: {
    name: "vm_rows_received_by_storage_total",
    hint: "Total metric rows received by storage for processing",
  },
  VM_ROWS_ADDED_TO_STORAGE_TOTAL: {
    name: "vm_rows_added_to_storage_total",
    hint: "Total metric rows successfully added to storage",
  },
  VM_DATA_SIZE_BYTES: {
    name: "vm_data_size_bytes",
    hint: "Total size of stored metrics",
  },
  VM_ROWS_MERGED_TOTAL: {
    name: "vm_rows_merged_total",
    hint: "Total metric rows merged during storage compaction.",
  },
  VM_ROWS_IGNORED_TOTAL: {
    name: "vm_rows_ignored_total",
    hint: "Total metric rows ignored (filtered out before storage)",
  },
  VM_ROWS_DELETED_TOTAL: {
    name: "vm_rows_deleted_total",
    hint: "Total metric rows deleted from storage",
  },
  VM_VMINSERT_CONNS: {
    name: "vm_vminsert_conns",
    hint: "Current active connections maintained by VMInsert",
  },
  VM_VMINSERT_CONN_ERRORS_TOTAL: {
    name: "vm_vminsert_conn_errors_total",
    hint: "Total connection errors encountered by VMInsert",
  },
  VM_VMINSERT_METRICS_READ_TOTAL: {
    name: "vm_vminsert_metrics_read_total",
    hint: "Total metric samples read by VMInsert.",
  },
  VM_VMSELECT_CONNS: {
    name: "vm_vmselect_conns",
    hint: "Current active connections to VMSelect",
  },
  VM_VMSELECT_CONN_ERRORS_TOTAL: {
    name: "vm_vmselect_conn_errors_total",
    hint: "Total connection errors encountered by VMSelect",
  },
  VM_VMSELECT_METRIC_ROWS_READ_TOTAL: {
    name: "vm_vmselect_metric_rows_read_total",
    hint: "Total metric rows read by VMSelect from storage",
  },
  VM_ZSTD_BLOCK_ORIGINAL_BYTES_TOTAL: {
    name: "vm_zstd_block_original_bytes_total",
    hint: "Uncompressed size processed in ZSTD blocks",
  },
  VM_ZSTD_BLOCK_COMPRESSED_BYTES_TOTAL: {
    name: "vm_zstd_block_compressed_bytes_total",
    hint: "Compressed size produced for ZSTD blocks",
  },
};
