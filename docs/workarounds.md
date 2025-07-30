# KOF Workarounds

These are temporary workarounds that don't need to be included to the main docs.

## Failed to create temporary file

If you're testing with `kind` cluster on Mac arm64,
and istio sidecars are crashing with `Failed to create temporary file` error,
then edit the `istio-sidecar-injector` ConfigMap and add there:

```yaml
            volumeMounts:
            - mountPath: /tmp
              name: tmp
...
          volumes:
          - emptyDir:
            name: tmp
```

Alternative is to disable Rosetta and degrade performance:
* https://github.com/istio/istio/issues/55655
* https://github.com/kgateway-dev/kgateway/issues/9800
