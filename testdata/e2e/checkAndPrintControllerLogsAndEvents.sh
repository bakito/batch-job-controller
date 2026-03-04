#!/bin/bash
set -euo pipefail

LABEL_SELECTOR='app=example-job-controller'

NAMESPACE=e2e-batch

echo "---- events (namespace/${NAMESPACE}) ----"
events="$(kubectl get events -n "${NAMESPACE}" --sort-by='.lastTimestamp')"
echo "${events}"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  echo 'Namespace Events' >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
  echo "${events}" >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
fi

pod="$(kubectl get pod -n "${NAMESPACE}" -l "${LABEL_SELECTOR}" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')"

echo "---- logs (pod/${pod}) ----"
logs="$(kubectl logs -n "${NAMESPACE}" "${pod}" --all-containers=true)"
echo -n "${logs}"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  echo 'Controller Pod logs' >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
  echo -n "${logs}" >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
fi

if echo "${logs}" | grep -q '"level":"error"'; then
  echo "💥 ERROR: Found error level logs in controller"
  exit 1
fi