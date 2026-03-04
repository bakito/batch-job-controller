#!/bin/bash
set -euo pipefail

LABEL_SELECTOR='app=example-job-controller'

NAMESPACE=e2e-batch

pod="$(kubectl get pod -n "${NAMESPACE}" -l "${LABEL_SELECTOR}" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')"

logs="$(kubectl logs -n "${NAMESPACE}" "${pod}" --all-containers=true)"
echo -n "${logs}"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  echo 'Controller Pod logs' >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
  echo -n "${logs}" >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
fi