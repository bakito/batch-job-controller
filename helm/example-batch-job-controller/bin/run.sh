#!/bin/bash
trap '>&2 echo ERROR: Command on line $LINENO failed: $(tail -n+$LINENO $0 | head -n1) && exit 1' ERR

echo "👷 starting job"
echo
echo "🔡 printing env"
env
echo "😴 sleep 10s"
sleep 10
echo "📞 calling report callback: ${CALLBACK_SERVICE_RESULT_URL}"

echo "- send file"
curl --silent --show-error -X POST -H 'Content-Disposition: attachment;filename="test.txt"' --data-binary 'This is an uploaded file' "${CALLBACK_SERVICE_FILE_URL}"

echo "- trigger event"
curl --silent --show-error -X POST -H "Content-Type: application/json; charset=utf-8" --data-binary '{"warning": false,"reason": "TestReason","message": "test message with %s","args": ["arg"]}' "${CALLBACK_SERVICE_EVENT_URL}"

echo "- send metric"
curl --silent --show-error -X POST -H "Content-Type: application/json; charset=utf-8" --data-binary '{ "my_metric": [{ "value": 1.0, "labels": { "label_a": "AAA", "label_b": "BBB" }}] }' "${CALLBACK_SERVICE_RESULT_URL}"

echo "🏁 done"
