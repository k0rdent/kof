#!/usr/bin/env python3
"""
VictoriaMetrics/VictoriaLogs migration script.

This script exports metrics/logs from VictoriaMetrics/VictoriaLogs, allows transformation,
and imports them back with streaming support.
"""

import argparse
import gzip
import io
import json
import os
import sys
import threading
from pathlib import Path
from typing import Callable, Optional

import requests
from tqdm import tqdm

# Check environment variable to skip SSL certificate verification
SKIP_SSL_VERIFY = os.getenv("SKIP_SSL_VERIFY", "").lower() in ("1", "true", "yes")


def identity_transform(line: dict) -> dict:
    """Default transformation function that returns the line unchanged."""
    return line

def open_file(file_name: str, mode: str = "rt", encoding: str = "utf-8"):
    # Determine if output should be gzipped
    compress = file_name.endswith(".gz")
    if compress:
        return gzip.open(file_name, mode, encoding=encoding)
    return open(file_name, mode, encoding=encoding)

def get_endpoints(migration_type: str, read_endpoint: str, write_endpoint: str):
    """
    Get export and import endpoints based on migration type.
    
    Args:
        migration_type: Either "metrics" or "logs"
        read_endpoint: VictoriaMetrics/VictoriaLogs read endpoint
        write_endpoint: VictoriaMetrics/VictoriaLogs write endpoint
        
    Returns:
        Tuple of (export_url, export_params, export_data, import_url)
    """
    if migration_type == "metrics":
        export_url = f"{read_endpoint}/api/v1/export"
        export_params = {"reduce_mem_usage": "1"}
        export_data = {"match[]": '{__name__!=""}'}
        import_url = f"{write_endpoint}".replace("/write", "/import")
        return export_url, export_params, export_data, import_url
    elif migration_type == "logs":
        export_url = f"{read_endpoint}/select/logsql/query"
        export_params = None
        export_data = {"query": '*'}
        import_url = f"{write_endpoint}".replace("/insert/opentelemetry/v1/logs","/insert/jsonline")
        return export_url, export_params, export_data, import_url
    else:
        raise ValueError(f"Unknown migration type: {migration_type}")

def export_data(
    migration_type: str,
    read_endpoint: str,
    username: Optional[str],
    password: Optional[str],
    output_file: str,
    progress_bar: tqdm,
    transform_fn: Optional[Callable[[dict], dict]] = None,
) -> None:
    """
    Export metrics/logs from VictoriaMetrics/VictoriaLogs API with optional on-the-fly transformation.
    
    Args:
        migration_type: Either "metrics" or "logs"
        read_endpoint: VictoriaMetrics/VictoriaLogs read endpoint
        username: Authentication username (optional)
        password: Authentication password (optional)
        output_file: Path to output file (supports .gz extension)
        transform_fn: Optional function to transform each JSON line (default: None)
        progress_bar: Progress bar to update
    """
    export_url, export_params, export_data, _ = get_endpoints(migration_type, read_endpoint, "")
    headers = {"Accept-Encoding": "gzip"}
    auth = (username, password) if username and password else None

    # Use identity transform if none provided
    if transform_fn is None:
        transform_fn = identity_transform

    try:
        # Stream the response
        request_kwargs = {
            "url": export_url,
            "data": export_data,
            "headers": headers,
            "auth": auth,
            "stream": True,
            "timeout": None,
            "verify": not SKIP_SSL_VERIFY,
        }
        if export_params:
            request_kwargs["params"] = export_params
        
        with requests.post(**request_kwargs) as response:
            response.raise_for_status()

            # If transformation is needed, decompress, transform, and re-compress
            if transform_fn != identity_transform:
                # Decompress gzip stream and parse JSON lines
                with gzip.GzipFile(fileobj=response.raw, mode="rb") as gzip_decompressor:
                    text_stream = io.TextIOWrapper(gzip_decompressor, encoding="utf-8")

                    with open_file(output_file, mode="wt") as output_fp:
                        line_count = 0
                        for line in text_stream:
                            line = line.strip()
                            if not line:
                                continue

                            try:
                                # Parse JSON line
                                json_obj = json.loads(line)
                                # Transform
                                transformed = transform_fn(json_obj)
                                # Write transformed line
                                output_fp.write(json.dumps(transformed) + "\n")
                                line_count += 1

                                progress_bar.update(1)
                            except json.JSONDecodeError as e:
                                print(
                                    f"Warning: Skipping invalid JSON line: {e}",
                                    file=sys.stderr,
                                )
                                continue

                        progress_bar.set_postfix({"lines": line_count, "status": "exported & transformed"})
            else:
                with open_file(output_file, mode="wb", encoding=None) as output_fp:
                    # Stream response content to file
                    for chunk in response.iter_content(chunk_size=8192):
                        if chunk:
                            output_fp.write(chunk)
                            progress_bar.update(len(chunk))

                    progress_bar.set_postfix({"status": "exported"})

    except requests.exceptions.RequestException as e:
        data_type = "logs" if migration_type == "logs" else "metrics"
        print(f"Error exporting {data_type}: {e}", file=sys.stderr)
        sys.exit(1)


def transform_data(
    input_file: str,
    output_file: str,
    transform_fn: Callable[[dict], dict],
    progress_bar: tqdm,
) -> None:
    """
    Transform metrics/logs from input file to output file.
    
    Args:
        input_file: Path to input file (supports .gz extension)
        output_file: Path to output file (supports .gz extension)
        transform_fn: Function to transform each JSON line
        progress_bar: Progress bar to update
    """

    try:
        with open_file(input_file, mode="rt") as input_fp:
            with open_file(output_file, mode="wt") as output_fp:
                line_count = 0
                for line in input_fp:
                    line = line.strip()
                    if not line:
                        continue

                    try:
                        # Parse JSON line
                        json_obj = json.loads(line)
                        # Transform
                        transformed = transform_fn(json_obj)
                        # Write transformed line
                        output_fp.write(json.dumps(transformed) + "\n")
                        line_count += 1
                        progress_bar.update(1)
                    except json.JSONDecodeError as e:
                        print(
                            f"Warning: Skipping invalid JSON line: {e}",
                            file=sys.stderr,
                        )
                        continue

                progress_bar.set_postfix({"lines": line_count})
    except IOError as e:
        print(f"Error transforming data: {e}", file=sys.stderr)
        sys.exit(1)


class ProgressFile:
    """File-like object wrapper for tracking upload progress."""

    def __init__(self, file_obj, progress_bar: tqdm):
        self.file_obj = file_obj
        self.progress_bar = progress_bar

    def read(self, size=-1):
        chunk = self.file_obj.read(size)
        if chunk:
            self.progress_bar.update(len(chunk))
        return chunk

    def __iter__(self):
        return self

    def __next__(self):
        chunk = self.file_obj.read(8192)
        if not chunk:
            raise StopIteration
        self.progress_bar.update(len(chunk))
        return chunk

    def close(self):
        return self.file_obj.close()


class StreamingTransformer:
    """Generator that transforms JSON lines as they stream through."""

    def __init__(
        self,
        source_iter,
        transform_fn: Callable[[dict], dict],
        progress_bar: tqdm,
    ):
        self.source_iter = source_iter
        self.transform_fn = transform_fn
        self.progress_bar = progress_bar

    def __iter__(self):
        return self

    def __next__(self):
        line = next(self.source_iter)
        if not line:
            raise StopIteration

        line = line.strip()
        if not line:
            return self.__next__()

        try:
            json_obj = json.loads(line)
            transformed = self.transform_fn(json_obj)
            transformed_line = json.dumps(transformed) + "\n"
            self.progress_bar.update(1)
            return transformed_line.encode("utf-8")
        except json.JSONDecodeError as e:
            print(
                f"Warning: Skipping invalid JSON line: {e}",
                file=sys.stderr,
            )
            return self.__next__()


class ChunkedWriter:
    """File-like object that writes data in chunks to a destination with progress tracking."""

    def __init__(self, destination, progress_bar: tqdm, chunk_size: int = 8192):
        self.destination = destination
        self.progress_bar = progress_bar
        self.chunk_size = chunk_size
        self.buffer = bytearray()

    def write(self, data):
        """Write data to destination in chunks."""
        if isinstance(data, str):
            data = data.encode("utf-8")
        
        self.buffer.extend(data)
        
        # Flush chunks when buffer reaches chunk_size
        while len(self.buffer) >= self.chunk_size:
            chunk = bytes(self.buffer[:self.chunk_size])
            self.buffer = self.buffer[self.chunk_size:]
            self.destination.write(chunk)
            self.progress_bar.update(len(chunk))

    def flush(self):
        """Flush remaining data."""
        if self.buffer:
            self.destination.write(bytes(self.buffer))
            self.progress_bar.update(len(self.buffer))
            self.buffer = bytearray()

    def close(self):
        """Flush remaining data (does not close destination)."""
        self.flush()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()


def stream_export_to_import(
    migration_type: str,
    read_endpoint: str,
    read_username: Optional[str],
    read_password: Optional[str],
    write_endpoint: str,
    write_username: Optional[str],
    write_password: Optional[str],
    transform_fn: Callable[[dict], dict],
    export_progress_bar: tqdm,
    import_progress_bar: tqdm,
) -> None:
    """
    Stream metrics/logs directly from export to import with transformation.
    
    Args:
        migration_type: Either "metrics" or "logs"
        read_endpoint: VictoriaMetrics/VictoriaLogs read endpoint
        read_username: Authentication username for read endpoint (optional)
        read_password: Authentication password for read endpoint (optional)
        write_endpoint: VictoriaMetrics/VictoriaLogs write endpoint
        write_username: Authentication username for write endpoint (optional)
        write_password: Authentication password for write endpoint (optional)
        transform_fn: Function to transform each JSON line
        export_progress_bar: Progress bar for export
        import_progress_bar: Progress bar for import
    """
    export_url, export_params, export_data, import_url = get_endpoints(migration_type, read_endpoint, write_endpoint)
    export_headers = {"Accept-Encoding": "gzip"}
    export_auth = (read_username, read_password) if read_username and read_password else None

    import_headers = {"Content-Encoding": "gzip"}
    import_auth = (write_username, write_password) if write_username and write_password else None

    try:
        # Use a pipe to connect export → transform → compress → import
        pipe_read_fd, pipe_write_fd = os.pipe()
        import_pipe_read = os.fdopen(pipe_read_fd, "rb")
        import_pipe_write = os.fdopen(pipe_write_fd, "wb")
        
        def write_compressed_data():
            """Write transformed and compressed data to the pipe."""
            try:
                # Stream export response
                request_kwargs = {
                    "url": export_url,
                    "data": export_data,
                    "headers": export_headers,
                    "auth": export_auth,
                    "stream": True,
                    "timeout": None,
                    "verify": not SKIP_SSL_VERIFY,
                }
                if export_params:
                    request_kwargs["params"] = export_params
                
                with requests.post(**request_kwargs) as export_response:
                    export_response.raise_for_status()

                    # Decompress gzip stream and parse JSON lines
                    with gzip.GzipFile(fileobj=export_response.raw, mode="rb") as gzip_decompressor:
                        text_stream = io.TextIOWrapper(gzip_decompressor, encoding="utf-8")
                        transformer = StreamingTransformer(text_stream, transform_fn, export_progress_bar)

                        # Write to pipe with chunking and gzip compression
                        with ChunkedWriter(import_pipe_write, import_progress_bar) as chunked_writer:
                            with gzip.GzipFile(fileobj=chunked_writer, mode="wb") as gzip_compressor:
                                line_count = 0
                                for transformed_bytes in transformer:
                                    gzip_compressor.write(transformed_bytes)
                                    line_count += 1
                                
                                export_progress_bar.set_postfix({"lines": line_count, "status": "exported"})
            except Exception as e:
                print(f"Error writing compressed data: {e}", file=sys.stderr)
                raise
            finally:
                import_pipe_write.close()

        # Start writer thread
        writer_thread = threading.Thread(target=write_compressed_data, daemon=True)
        writer_thread.start()

        # Stream from pipe to import endpoint
        try:
            import_response = requests.post(
                import_url,
                data=import_pipe_read,
                headers=import_headers,
                auth=import_auth,
                timeout=None,
                verify=not SKIP_SSL_VERIFY,
            )
            import_response.raise_for_status()

            import_progress_bar.set_postfix({"status": "imported"})
        finally:
            import_pipe_read.close()
            writer_thread.join(timeout=300)  # Increased timeout for large migrations
                
    except requests.exceptions.RequestException as e:
        data_type = "logs" if migration_type == "logs" else "metrics"
        print(f"Error during streaming migration ({data_type}): {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error during transformation: {e}", file=sys.stderr)
        sys.exit(1)


def import_data(
    migration_type: str,
    write_endpoint: str,
    username: Optional[str],
    password: Optional[str],
    input_file: str,
    progress_bar: tqdm,
) -> None:
    """
    Import metrics/logs to VictoriaMetrics/VictoriaLogs API with streaming support.
    
    Args:
        migration_type: Either "metrics" or "logs"
        write_endpoint: VictoriaMetrics/VictoriaLogs write endpoint
        username: Authentication username (optional)
        password: Authentication password (optional)
        input_file: Path to input file (supports .gz extension)
        progress_bar: Progress bar to update
    """
    _, _, _, import_url = get_endpoints(migration_type, "", write_endpoint)
    headers = {}
    auth = (username, password) if username and password else None

    # Determine if input is gzipped
    input_compress = input_file.endswith(".gz")
    if input_compress:
        headers["Content-Encoding"] = "gzip"

    try:
        with open(input_file, mode="rb") as input_fp:
            # Wrap file object to track progress
            progress_file = ProgressFile(input_fp, progress_bar)

            # Stream the file content with progress tracking
            response = requests.post(
                import_url,
                data=progress_file,
                headers=headers,
                auth=auth,
                timeout=None,
                verify=not SKIP_SSL_VERIFY,
            )
            response.raise_for_status()

            progress_bar.set_postfix({"status": "imported"})
    except (IOError, requests.exceptions.RequestException) as e:
        data_type = "logs" if migration_type == "logs" else "metrics"
        print(f"Error importing {data_type}: {e}", file=sys.stderr)
        sys.exit(1)


def main():
    """Main function to run the migration workflow."""
    parser = argparse.ArgumentParser(
        description="Migrate metrics/logs from VictoriaMetrics/VictoriaLogs to VictoriaMetrics/VictoriaLogs with transformation support"
    )
    parser.add_argument(
        "--type",
        choices=["metrics", "logs"],
        required=True,
        help="Type of data to migrate: 'metrics' or 'logs'",
    )
    parser.add_argument(
        "--read-endpoint",
        help="VictoriaMetrics/VictoriaLogs read endpoint (e.g., https://vm-read.example.com). If not specified, export step is skipped.",
    )
    parser.add_argument(
        "--write-endpoint",
        help="VictoriaMetrics/VictoriaLogs write endpoint (e.g., https://vm-write.example.com). If not specified, import step is skipped.",
    )
    parser.add_argument(
        "--read-username",
        help="Authentication username for read endpoint (optional if endpoint doesn't require auth)",
    )
    parser.add_argument(
        "--write-username",
        help="Authentication username for write endpoint (optional if endpoint doesn't require auth)",
    )
    parser.add_argument(
        "--export-file",
        help="Path to export file (default: victoria-metrics-export.jsonl.gz for metrics, victoria-logs-export.jsonl.gz for logs)",
    )
    parser.add_argument(
        "--transform-module",
        help="Python module path containing transform function (e.g., mymodule)",
    )
    parser.add_argument(
        "--transform-function",
        default="transform",
        help="Name of transform function (default: transform)",
    )
    parser.add_argument(
        "--stream",
        action="store_true",
        help="Stream directly from export to import without storing files",
    )

    args = parser.parse_args()
    migration_type = args.type

    # Set default export file based on type
    if not args.export_file:
        if migration_type == "metrics":
            args.export_file = "victoria-metrics-export.jsonl.gz"
        else:
            args.export_file = "victoria-logs-export.jsonl.gz"

    # Get password environment variables
    read_password = os.getenv("READ_PASSWORD")
    write_password = os.getenv("WRITE_PASSWORD")

    # Credentials are optional - use None if not provided
    read_username = args.read_username
    write_username = args.write_username

    # Load transformation function if provided
    transform_fn = identity_transform
    if args.transform_module:
        try:
            module = __import__(args.transform_module, fromlist=[args.transform_function])
            transform_fn = getattr(module, args.transform_function)
        except (ImportError, AttributeError, ValueError) as e:
            print(
                f"Error loading transform function: {e}",
                file=sys.stderr,
            )
            sys.exit(1)

    data_type = migration_type  # For user-facing messages

    # Stream mode: direct export to import
    if args.stream:
        if not args.read_endpoint:
            print(f"Error: --read-endpoint is required for streaming mode", file=sys.stderr)
            sys.exit(1)
        if not args.write_endpoint:
            print(f"Error: --write-endpoint is required for streaming mode", file=sys.stderr)
            sys.exit(1)
        print(f"Streaming {data_type} from export to import...")
        with tqdm(
            unit="lines",
            desc="Exporting & Transforming",
            dynamic_ncols=True,
        ) as export_pbar, tqdm(
            unit="B",
            unit_scale=True,
            desc="Importing",
            dynamic_ncols=True,
        ) as import_pbar:
            stream_export_to_import(
                migration_type,
                args.read_endpoint,
                read_username,
                read_password,
                args.write_endpoint,
                write_username,
                write_password,
                transform_fn,
                export_progress_bar=export_pbar,
                import_progress_bar=import_pbar,
            )
        print("Migration completed successfully!")
        return

    # File-based mode: Step 1: Export data
    if args.read_endpoint:
        print(f"Exporting {data_type}...")
        # Use lines progress bar if transformation is enabled
        if transform_fn != identity_transform:
            pbar = tqdm(
                unit="lines",
                desc="Exporting & Transforming",
                dynamic_ncols=True,
            )
        else:
            pbar = tqdm(
                unit="B",
                unit_scale=True,
                desc="Exporting",
                dynamic_ncols=True,
            )
        with pbar:
            export_data(
                migration_type,
                args.read_endpoint,
                read_username,
                read_password,
                args.export_file,
                pbar,
                transform_fn=transform_fn,
            )
        print(f"{data_type.capitalize()} exported to {args.export_file}")
    else:
        print(f"Skipping export, using existing file: {args.export_file}")

    # Transformation is now done during export, no separate step needed
    # Step 2: Import data
    if args.write_endpoint:
        print(f"Importing {data_type}...")
        # Get file size for progress
        try:
            file_size = Path(args.export_file).stat().st_size
        except OSError as e:
            print(f"Error getting file size: {e}", file=sys.stderr)
            sys.exit(1)

        with tqdm(
            total=file_size,
            unit="B",
            unit_scale=True,
            desc="Importing",
            dynamic_ncols=True,
        ) as pbar:
            import_data(
                migration_type,
                args.write_endpoint,
                write_username,
                write_password,
                args.export_file,
                progress_bar=pbar,
            )
        print(f"{data_type.capitalize()} imported successfully")
    else:
        print("Skipping import")

    print("Migration completed successfully!")


if __name__ == "__main__":
    main()

