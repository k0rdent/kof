# Trace Monitoring System

## Overview

The Trace Monitoring System enables effective collection, management, and visualization of distributed traces in microservices architectures. It allows to monitor request flows across services, identify performance bottlenecks, and analyze service dependencies. The system primarily comprises two components:

* OpenTelemetry Operator: Automates the instrumentation of applications within Kubernetes clusters.
* Jaeger: An open-source, end-to-end distributed tracing system for collecting, storing, and visualizing trace data.

## Collecting Traces

To collect traces, implement automatic instrumentation for your applications. This process involves adding specific annotations to your applications, enabling the OpenTelemetry Operator to inject the instrumentation agents.

### Annotating Applications for Instrumentation

To enable automatic instrumentation, add the appropriate annotation to your applications manifest. For example, for a Python application, the deployment manifest would include:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: python-app
  namespace: kof
spec:
  replicas: 1
  selector:
    matchLabels:
      app: python-app
  template:
    metadata:
      labels:
        app: python-app
      annotations:
        instrumentation.opentelemetry.io/inject-python: "true"
    spec:
      containers:
        - name: app
          image: python:3.9-slim
          ports:
            - containerPort: 8080 

```

The annotation `instrumentation.opentelemetry.io/inject-python: "true"` directs the OpenTelemetry Operator to inject the Python instrumentation agent into your application.

### Supported Languages and Corresponding Annotations

The OpenTelemetry Operator supports automatic instrumentation for various languages. Below are the annotations for each:

* Java: `instrumentation.opentelemetry.io/inject-java: "true"`
* Node.js: `instrumentation.opentelemetry.io/inject-nodejs: "true"`
* Python: `instrumentation.opentelemetry.io/inject-python: "true"`
* .NET: `instrumentation.opentelemetry.io/inject-dotnet: "true"`
* Go: `instrumentation.opentelemetry.io/inject-go: "true"`

For Go applications, an additional annotation specifying the path to the executable is required:

```yaml
instrumentation.opentelemetry.io/otel-go-auto-target-exe: "/path/to/executable"
```

Ensure that the Instrumentation resource is deployed in the same namespace as your application before applying these annotations. For more details, refer to the official [OpenTelemetry documentation](https://opentelemetry.io/docs/platforms/kubernetes/operator/automatic/).

## Testing Locally

Before beginning testing, ensure that you have a [development cluster with KOF installed](https://github.com/k0rdent/kof/blob/main/docs/DEV.md). To test the trace system locally, apply the following manifest to deploy the service and pods:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: test-server-svc
  namespace: kof
spec:
  selector:
    app: test-server
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
---
apiVersion: v1
kind: Pod
metadata:
  name: test-server
  namespace: kof
  labels:
    app: test-server
  annotations:
    instrumentation.opentelemetry.io/inject-python: "true"
spec:
  containers:
    - name: server
      image: python:3.9-slim
      command:
        - /bin/sh
        - -c
        - |
          pip install flask && \
          python - <<'EOF'
          from flask import Flask
          app = Flask(__name__)
          @app.route("/")
          def hello():
              return "Hello World!"
          if __name__ == "__main__":
              app.run(host="0.0.0.0", port=8080)
          EOF
---
apiVersion: v1
kind: Pod
metadata:
  name: test-client
  namespace: kof
spec:
  containers:
    - name: client
      image: curlimages/curl:7.85.0
      command:
        [
          "sh",
          "-c",
          "sleep 5; curl http://test-server-svc.your-namespace.svc.cluster.local && sleep 3600",
        ] 
```

This manifest deploys a simple Flask server (test-server) and a client (test-client) that makes a request to the server. The server pod includes the annotation to enable Python auto-instrumentation.

After deploying these resources, set up port-forwarding to access Jaeger UI:

```zsh
kubectl port-forward svc/kof-collectors-jaeger-query 16686:16686 -n kof
```

Once port-forwarding is established, navigate to `http://localhost:16686` in your browser to access the Jaeger UI and verify that traces from the `test-server` are being collected.
