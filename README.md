# Velero Plugin for Service Selector Updates

A Velero plugin that allows you to update service selectors when restoring from backup.

## Overview

When restoring applications using Velero, you might want to update service selectors to match new deployment names or labels. This plugin enables you to modify service selectors during restore.

## Deploying the plugin

The plugin is available as a container image from GitHub Container Registry:
ghcr.io/eth-eks/velero-plugin-update-replicas:latest

To deploy your plugin image to a Velero server:

### Using Velero CLI
1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Run `velero plugin add <registry/image:version>`. Example with a dockerhub image: `velero plugin add velero/velero-plugin-example`.

### Using Helm
1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Add the plugin to your Velero Helm chart's `values.yaml`:

    ```yaml
    velero:
      initContainers:
        - name: velero-plugin-update-replicas
              image: ghcr.io/eth-eks/velero-plugin-update-replicas:latest
              imagePullPolicy: Always
              volumeMounts:
                - name: plugins
                  mountPath: /target
    ```

## Usage

3. When restoring, the plugin will automatically set the replica count according to the annotation:

```bash
velero restore create --from-backup my-backup
```

## Supported Resources

- Deployments
- StatefulSets

## How It Works

The plugin:
1. Intercepts resources during restore
2. Checks for the `eth-eks.velero/replicas-value-after-recovery` annotation
3. If present, updates the replica count to the specified value
4. If absent or invalid, maintains the original replica count

## Development

### Prerequisites

- Go 1.23 or later
- Docker

### Running Tests

```bash
go test -v ./...
```

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
