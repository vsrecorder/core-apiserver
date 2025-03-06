#!/bin/bash

if [ -z "${SLACK_WEBHOOK_URL}" ] || \
   [ -z "${SUCCESS}"] || [ -z "${WORKFLOW}" ] || [ -z "${SERVER_URL}" ] || [ -z "${REPOSITORY}" ] || \
   [ -z "${BRANCH}" ] || [ -z "${SHA}" ] || [ -z "${RUN_ID}" ] || [ -z "${ACTOR}" ] \
   ; then
    echo "実行に必要な環境変数が設定されていません。"
    exit 1
fi

if [ "${SUCCESS}" = true ]; then
    STATUS_COLOR="#36a64f"
    STATUS_ICON=":white_check_mark:"
    STATUS_TEXT="Success"
else
    STATUS_COLOR="#C04040"
    STATUS_ICON=":x:"
    STATUS_TEXT="Failure"
fi

_COMMIT="<${SERVER_URL}/${REPOSITORY}/commit/${SHA}|${SHA::8}>"
_ACTIONS="<${SERVER_URL}/${REPOSITORY}/actions/runs/${RUN_ID}|${WORKFLOW}>"
_REPOSITORY="<${SERVER_URL}/${REPOSITORY}/commits/${BRANCH}|${REPOSITORY}>"

curl -X POST \
    -H "Content-type: application/json" \
    --data "{\"attachments\": [{
        \"color\": \"$STATUS_COLOR\",
        \"blocks\": [{
            \"type\": \"section\",
            \"text\": {
                \"type\": \"mrkdwn\",
                \"text\": \"${STATUS_ICON} ${STATUS_TEXT}: ${_ACTIONS}\n:memo: Commit: ${_COMMIT}\n:github: Repository: ${_REPOSITORY}\n:bust_in_silhouette:  Author: ${ACTOR}\"
            }
        }]
    }]}" \
    $SLACK_WEBHOOK_URL
