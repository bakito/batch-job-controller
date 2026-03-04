#!/bin/bash

#!/usr/bin/env bash
set -euo pipefail

LABEL_SELECTOR='batch-job-controller.bakito.github.com/owner=example-job-controller'

NAMESPACE=e2e-batch

FIND_TIMEOUT_SECONDS=90          # 1.5 min
COMPLETE_TIMEOUT_SECONDS=600     # adjust if your pod can run longer

deadline=$((SECONDS + FIND_TIMEOUT_SECONDS))

pod=""

echo "Waiting up to ${FIND_TIMEOUT_SECONDS}s for a pod with label: ${LABEL_SELECTOR} (ns=${NAMESPACE})"

while (( SECONDS < deadline )); do
  # Pick the newest pod if there are multiple
  pod="$(
    kubectl get pod -n "${NAMESPACE}" -l "${LABEL_SELECTOR}" \
      --sort-by=.metadata.creationTimestamp \
      -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
    | tail -n 1
  )" || true

  if [[ -n "${pod}" ]]; then
    echo "Found pod: ${pod}"
    break
  fi

  sleep 2
done

if [[ -z "${pod}" ]]; then
  echo "ERROR: No pod found within ${FIND_TIMEOUT_SECONDS}s for label ${LABEL_SELECTOR} in namespace ${NAMESPACE}" >&2
  exit 1
fi

phase="$(kubectl get pod -n "${NAMESPACE}" "${pod}" -o jsonpath='{.status.phase}')"
echo "Current phase: ${phase}"

if [[ "${phase}" != "Succeeded" && "${phase}" != "Failed" ]]; then
  echo "Waiting for pod to complete (Succeeded/Failed), timeout=${COMPLETE_TIMEOUT_SECONDS}s..."
  complete_deadline=$((SECONDS + COMPLETE_TIMEOUT_SECONDS))

  while (( SECONDS < complete_deadline )); do
    phase="$(kubectl get pod -n "${NAMESPACE}" "${pod}" -o jsonpath='{.status.phase}')"
    if [[ "${phase}" == "Succeeded" || "${phase}" == "Failed" ]]; then
      break
    fi
    sleep 2
  done
fi

phase="$(kubectl get pod -n "${NAMESPACE}" "${pod}" -o jsonpath='{.status.phase}')"
echo "Final phase: ${phase}"
echo "---- logs (pod/${pod}) ----"
logs="$(kubectl logs -n "${NAMESPACE}" "${pod}" --all-containers=true)"
echo -n "${logs}"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  echo 'Job Pod logs' >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
  echo -n "${logs}" >> $GITHUB_STEP_SUMMARY
  echo '```' >> $GITHUB_STEP_SUMMARY
fi

if [[ "${phase}" == "Failed" ]]; then
  echo "ERROR: Pod failed: ${pod}" >&2
  exit 2
fi