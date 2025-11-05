# VictoriaMetrics/VictoriaLogs Migration Script

A unified Python script for migrating metrics and logs from one VictoriaMetrics/VictoriaLogs instance to another with support for transformation and streaming.

## Features

- **Support for both metrics and logs**: Single script handles both VictoriaMetrics metrics and VictoriaLogs migrations
- **Streaming mode**: Direct export-to-import without storing intermediate files (low memory usage)
- **File-based mode**: Export (with optional transformation) and import with intermediate file storage
- **On-the-fly transformation**: Apply custom transformations to metrics/logs during export
- **Progress tracking**: Real-time progress bars for export/transformation and import operations
- **Gzip compression**: Automatic compression/decompression support
- **Flexible credentials**: Separate authentication for read and write endpoints

## Prerequisites

- Python 3.6 or higher
- Required Python packages (install via `pip install -r ../requirements.txt`):
  - `requests`
  - `tqdm`

## Installation

From the repository root:

```bash
pip install -r scripts/requirements.txt
```

## Usage

### Basic Migration (Streaming Mode)

Stream data directly from source to destination without storing files:

```bash
export READ_PASSWORD="source_pass"
export WRITE_PASSWORD="dest_pass"

python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-source.example.com \
  --write-endpoint https://vm-dest.example.com \
  --read-username source_user \
  --write-username dest_user \
  --stream
```

For logs, use `--type logs` and VictoriaLogs endpoints.

### File-Based Migration

Export (with optional transformation) and import with intermediate file storage. Transformation is applied during export and saved to the export file:

```bash
# Full workflow (export with transformation, then import)
export READ_PASSWORD="source_pass"
export WRITE_PASSWORD="dest_pass"

python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-source.example.com \
  --write-endpoint https://vm-dest.example.com \
  --read-username source_user \
  --write-username dest_user

# Skip export, import only (omit --read-endpoint)
export WRITE_PASSWORD="dest_pass"
python scripts/victoria-migration/migration.py \
  --type metrics \
  --write-endpoint https://vm-dest.example.com \
  --write-username dest_user

# Export only (omit --write-endpoint)
export READ_PASSWORD="source_pass"
python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-source.example.com \
  --read-username source_user

# Custom export file
python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-source.example.com \
  --write-endpoint https://vm-dest.example.com \
  --read-username source_user \
  --write-username dest_user \
  --export-file custom-filename.gz
```

For logs, use `--type logs` and VictoriaLogs endpoints.

### Migration with Transformation

Apply custom transformations to metrics/logs during migration. **In file-based mode, transformation happens during export and the transformed data is saved to the export file**. In streaming mode, transformation happens on-the-fly between export and import.

#### 1. Create a transformation module

Create a Python file (e.g., `mytransform.py`) with a transformation function:

**For Metrics:**

```python
def transform(json_line: dict) -> dict:
    """
    Transform a single metric JSON line.
    
    Args:
        json_line: Dictionary with 'metric', 'values', and 'timestamps' keys
        
    Returns:
        Transformed dictionary
    """
    # Example: Remove specific labels
    if 'metric' in json_line:
        metric = json_line['metric']
        metric.pop('vm_account_id', None)
        metric.pop('vm_project_id', None)
    
    return json_line
```

**For Logs:**

```python
def transform(json_line: dict) -> dict:
    """
    Transform a single log JSON line.
    
    Args:
        json_line: Dictionary with log fields (e.g., '_time', '_msg', '_stream', etc.)
        
    Returns:
        Transformed dictionary
    """
    # Example: Modify log fields
    if '_stream' in json_line:
        # Remove or modify stream labels as needed
        json_line['_stream'].pop('unwanted_label', None)
    
    return json_line
```

#### 2. Use the transformation

```bash
# Streaming mode
export READ_PASSWORD="source_pass"
export WRITE_PASSWORD="dest_pass"

python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-source.example.com \
  --write-endpoint https://vm-dest.example.com \
  --read-username source_user \
  --write-username dest_user \
  --transform-module mytransform \
  --stream

# File-based mode
python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-source.example.com \
  --write-endpoint https://vm-dest.example.com \
  --read-username source_user \
  --write-username dest_user \
  --transform-module mytransform
```

If your transformation function has a different name:

```bash
python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-source.example.com \
  --write-endpoint https://vm-dest.example.com \
  --read-username source_user \
  --write-username dest_user \
  --transform-module mytransform \
  --transform-function custom_transform \
  --stream
```

## Command-Line Arguments

### Required Arguments

- `--type`: Type of data to migrate. Must be either `metrics` or `logs`.

**For Streaming Mode (`--stream`):**
- `--read-endpoint`: VictoriaMetrics/VictoriaLogs read endpoint URL (required in streaming mode)
- `--write-endpoint`: VictoriaMetrics/VictoriaLogs write endpoint URL (required in streaming mode)

**For File-Based Mode:**
- At least one of `--read-endpoint` or `--write-endpoint` must be specified
- If `--read-endpoint` is not specified, export is skipped (uses existing export file)
- If `--write-endpoint` is not specified, import is skipped

### Optional Arguments

- `--read-endpoint`: VictoriaMetrics/VictoriaLogs read endpoint URL. If not specified in file-based mode, export is skipped.
- `--write-endpoint`: VictoriaMetrics/VictoriaLogs write endpoint URL. If not specified in file-based mode, import is skipped.
- `--read-username`: Authentication username for the read endpoint (optional if endpoint doesn't require auth)
- `--write-username`: Authentication username for the write endpoint (optional if endpoint doesn't require auth)
- `--stream`: Enable streaming mode (direct export-to-import without file storage). Both endpoints are required.
- `--export-file`: Path for export file in file-based mode. If transformation is used, this file contains the transformed data. Default: `victoria-metrics-export.jsonl.gz` for metrics, `victoria-logs-export.jsonl.gz` for logs.
- `--transform-module`: Python module path containing transform function (e.g., `mymodule`).
- `--transform-function`: Name of transform function (default: `transform`)

### Environment Variables

- `READ_PASSWORD`: Authentication password for the read endpoint (optional if endpoint doesn't require auth)
- `WRITE_PASSWORD`: Authentication password for the write endpoint (optional if endpoint doesn't require auth)

## Modes of Operation

### Streaming Mode (`--stream`)

**Advantages:**
- No disk space required for intermediate files
- Lower memory usage for large datasets
- Faster overall migration

**When to use:**
- Large datasets where disk space is a concern
- Direct migration without need for intermediate files
- When you have reliable network connections

### File-Based Mode (default)

**Advantages:**
- Can resume migration if interrupted (omit `--read-endpoint` to skip export, use existing file)
- Can inspect/validate exported data before importing
- Can backup exported data
- Transformation is applied during export and saved to the export file
- Flexible: export only, import only, or both

**When to use:**
- Need to inspect or validate exported data
- Unreliable network connections
- Need to back up exported data
- Want to export once and import multiple times (to different destinations)
- Need to export or import separately

**How to skip steps:**
- Skip export: Omit `--read-endpoint` (script will use existing export file)
- Skip import: Omit `--write-endpoint` (script will only export to file)

## Enhancement Details

### Transformation During Export

In file-based mode, when a transformation function is provided, data is transformed on-the-fly during export. The transformed data is saved directly to the export file, eliminating the need for a separate transformed file. This approach:

- Reduces disk I/O operations
- Saves disk space (no intermediate transformed file)
- Simplifies the workflow (only one file to manage)

### Output Format

**Metrics format:**

The script exports metrics in VictoriaMetrics JSON Lines format:

```json
{"metric":{"__name__":"metric_name","label1":"value1"},"values":[1,2,3],"timestamps":[1000,2000,3000]}
{"metric":{"__name__":"metric_name","label2":"value2"},"values":[4,5,6],"timestamps":[4000,5000,6000]}
```

Each line is a complete JSON object representing a single metric series. If transformation is applied, the export file contains the transformed metrics.

**Logs format:**

The script exports logs in VictoriaLogs JSON Lines format:

```json
{"_time":"2024-01-01T00:00:00Z","_msg":"log message","_stream":{"cluster":"prod","namespace":"default"}}
{"_time":"2024-01-01T00:00:01Z","_msg":"another message","_stream":{"cluster":"prod","namespace":"kube-system"}}
```

Each line is a complete JSON object representing a single log entry. If transformation is applied, the export file contains the transformed logs.

## Examples

### Example 1: Migration with Custom Export File

Use a custom file path for the export file (which will contain transformed data if transformation is enabled):

```bash
export READ_PASSWORD="secret"
export WRITE_PASSWORD="secret"

python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-old.example.com \
  --write-endpoint https://vm-new.example.com \
  --read-username admin \
  --write-username admin \
  --export-file /tmp/my-export.jsonl.gz
```

### Example 2: Skip Export or Import

Skip export and import from existing file:

```bash
# Import only (skip export - use existing export file)
export WRITE_PASSWORD="secret"

python scripts/victoria-migration/migration.py \
  --type metrics \
  --write-endpoint https://vm-new.example.com \
  --write-username admin
```

Export only (skip import - just export to file):

```bash
export READ_PASSWORD="secret"

python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-old.example.com \
  --read-username admin \
  --export-file /tmp/backup.jsonl.gz
```

### Example 3: Transformation - Remove Labels (Metrics)

Create `remove_labels.py`:

```python
def transform(json_line: dict) -> dict:
    """Remove vm_account_id and vm_project_id labels."""
    if 'metric' in json_line:
        json_line['metric'].pop('vm_account_id', None)
        json_line['metric'].pop('vm_project_id', None)
    return json_line
```

Run migration:

```bash
export READ_PASSWORD="secret"
export WRITE_PASSWORD="secret"

python scripts/victoria-migration/migration.py \
  --type metrics \
  --read-endpoint https://vm-old.example.com \
  --write-endpoint https://vm-new.example.com \
  --read-username admin \
  --write-username admin \
  --transform-module remove_labels \
  --stream
```

### Example 4: Transformation - Rename Clusters (Logs)

Create `rename_clusters.py`:

```python
def transform(json_line: dict) -> dict:
    """Rename cluster labels in log streams using a mapping."""
    if '_stream' in json_line and 'cluster' in json_line['_stream']:
        cluster = json_line['_stream']['cluster']
        cluster_map = {
            'prod-cluster': 'production',
            'dev-cluster': 'development'
        }
        if cluster in cluster_map:
            json_line['_stream']['cluster'] = cluster_map[cluster]
    return json_line
```

Run migration:

```bash
export READ_PASSWORD="secret"
export WRITE_PASSWORD="secret"

python scripts/victoria-migration/migration.py \
  --type logs \
  --read-endpoint https://vl-old.example.com \
  --write-endpoint https://vl-new.example.com \
  --read-username admin \
  --write-username admin \
  --transform-module rename_clusters \
  --stream
```

## Troubleshooting

### Authentication Errors

If you encounter authentication errors:

1. Verify endpoint URLs are correct
2. Check username/password credentials
3. Ensure endpoints support basic authentication
4. Verify you're using the correct environment variables (`READ_PASSWORD` and `WRITE_PASSWORD`)

### Network Timeouts

For large migrations:

1. Use file-based mode and resume on failure by omitting `--read-endpoint` to re-import existing export file
2. Check network stability
3. Consider breaking migration into smaller batches

### Memory Issues

If running out of memory:

1. Use `--stream` mode for lower memory usage
2. Process smaller time ranges if possible
3. Increase system resources

### Invalid JSON Lines

The script will skip invalid JSON lines and print warnings. Check stderr output for details.

### Gzip Issues

If gzip compression/decompression fails:

1. Ensure files have `.gz` extension for automatic compression
2. Check disk space availability
3. Verify file permissions

### Wrong Migration Type

If you get errors about endpoints or data format:

1. Verify you're using `--type metrics` for VictoriaMetrics endpoints
2. Verify you're using `--type logs` for VictoriaLogs endpoints
3. Check that the endpoints match the type you're migrating

## Security Considerations

- Passwords are read from environment variables (`READ_PASSWORD` and `WRITE_PASSWORD`) to avoid exposing them in command line or process lists
- Use HTTPS endpoints to encrypt data in transit
- Store intermediate files securely if they contain sensitive data
- Be cautious when setting environment variables in scripts or shared environments

## License

[Add appropriate license information based on your project]

