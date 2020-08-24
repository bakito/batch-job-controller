#!/bin/bash
trap '>&2 echo ERROR: Command on line $LINENO failed: $(tail -n+$LINENO $0 | head -n1) && exit 1' ERR

echo "starting"
env
sleep 10
echo "calling report callback: ${CALLBACK_SERVICE_RESULT_URL}"



curl --silent --show-error -X POST -H 'Content-Disposition: attachment;filename="test.txt"' --data-binary 'This is an uploaded file' "${CALLBACK_SERVICE_FILE_URL}"
curl --silent --show-error -X POST -H "Content-Type: application/json; charset=utf-8" --data-binary '{"warning": false,"reason": "TestReason","message": "test message with %s","args": ["arg"]}' "${CALLBACK_SERVICE_EVENT_URL}"
curl --silent --show-error -X POST -H "Content-Type: application/json; charset=utf-8" --data-binary '{ "my_metric": [{ "value": 1.0, "labels": { "label_a": "AAA", "label_b": "BBB" }}] }' "${CALLBACK_SERVICE_RESULT_URL}"

echo "done"
