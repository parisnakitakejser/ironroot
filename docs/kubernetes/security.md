# Kubernetes Security

The Helm chart defaults to non-root execution, dropped capabilities, ClusterIP Service, and mounted CA material. Enable NetworkPolicy and restrict Secret access in production.

The Root CA private key should never be copied into the cluster.
