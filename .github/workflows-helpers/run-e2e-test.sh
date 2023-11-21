# Check if the pod with the matching image SHA and purpose is ready
echo "Checking for ready sequencer pod with image SHA ${IMAGE_TAG}..."
while :; do
    # Get all pods with the matching purpose
    PODS_JSON=$(kubectl get pods -n ${NAMESPACE} -l pokt.network/purpose=sequencer -o json)

    # Check if any pods are running and have the correct image SHA
    READY_POD=$(echo $PODS_JSON | jq -r ".items[] | select(.status.phase == \"Running\") | select(.spec.containers[].image | contains(\"${IMAGE_TAG}\")) | .metadata.name")

    if [[ -n "${READY_POD}" ]]; then
        echo "Ready pod found: ${READY_POD}"
        break
    else
        echo "Sequencer with with an image ${IMAGE_TAG} is not ready yet. Will retry in 10 seconds..."
        sleep 10
    fi
done

# Check we can reach the sequencer endpoint
HTTP_STATUS=$(curl -s -o /dev/null -w '%{http_code}' http://${NAMESPACE}-sequencer:36657)
if [[ "${HTTP_STATUS}" -eq 200 ]]; then
    echo "HTTP request to devnet-issue-198-sequencer returned 200 OK."
else
    echo "HTTP request to devnet-issue-198-sequencer did not return 200 OK. Status code: ${HTTP_STATUS}. Retrying in 10 seconds..."
    sleep 10
fi

# Create a job to run the e2e tests
envsubst <.github/workflows-helpers/run-e2e-test-job-template.yaml >job.yaml
kubectl apply -f job.yaml

# Wait for the pod to be created and be in a running state
echo "Waiting for the pod to be in the running state..."
while :; do
    POD_NAME=$(kubectl get pods -n ${NAMESPACE} --selector=job-name=${JOB_NAME} -o jsonpath='{.items[*].metadata.name}')
    [[ -z "${POD_NAME}" ]] && echo "Waiting for pod to be scheduled..." && sleep 5 && continue
    POD_STATUS=$(kubectl get pod ${POD_NAME} -n ${NAMESPACE} -o jsonpath='{.status.phase}')
    [[ "${POD_STATUS}" == "Running" ]] && break
    echo "Current pod status: ${POD_STATUS}"
    sleep 5
done

echo "Pod is running. Monitoring logs and status..."
# Stream the pod logs in the background
kubectl logs -f ${POD_NAME} -n ${NAMESPACE} &

# Monitor pod status in a loop
while :; do
    CURRENT_STATUS=$(kubectl get pod ${POD_NAME} -n ${NAMESPACE} -o jsonpath="{.status.containerStatuses[0].state}")
    if echo $CURRENT_STATUS | grep -q 'terminated'; then
        EXIT_CODE=$(echo $CURRENT_STATUS | jq '.terminated.exitCode')
        if [[ "$EXIT_CODE" != "0" ]]; then
            echo "Container terminated with exit code ${EXIT_CODE}"
            kubectl delete job ${JOB_NAME} -n ${NAMESPACE}
            exit 1
        fi
        break
    fi
    sleep 5
done

# If the loop exits without failure, the job succeeded
echo "Job completed successfully"
kubectl delete job ${JOB_NAME} -n ${NAMESPACE}
