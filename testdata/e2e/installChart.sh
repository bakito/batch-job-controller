#!/bin/bash
set -e

NAMESPACE=e2e-batch

kubectl create ns "${NAMESPACE}" || true

if ! helm upgrade --install e2e-batch helm/example-batch-job-controller \
  --namespace "${NAMESPACE}" \
  -f testdata/e2e/values-e2e.yaml \
  --rollback-on-failure
then
  echo "ERROR: helm upgrade failed. Dumping diagnostics for namespace: ${NAMESPACE}" >&2
  echo "--- Pods (wide) ---" >&2
  kubectl get pods -n "${NAMESPACE}" -o wide >&2 || true
  echo "--- Events (sorted by lastTimestamp) ---" >&2
  kubectl get events -n "${NAMESPACE}" --sort-by='.lastTimestamp' >&2 || true
  exit 1
fi