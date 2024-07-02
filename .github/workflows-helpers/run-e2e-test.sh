# Log environment variables for debugging
echo "Environment variables:"
echo "NAMESPACE: ${NAMESPACE}"
echo "IMAGE_TAG: ${IMAGE_TAG}"

# Check if the pod with the matching image SHA and purpose is ready or needs recreation
echo "Checking for ready validator pod with image SHA ${IMAGE_TAG} or pods needing recreation..."
while :; do
    # Log the command
    echo "Running kubectl command to get pods with matching purpose=validator:"

    # Check if any pods are scheduled and have the correct image SHA
    SCHEDULED_POD=$(kubectl get pods -n "${NAMESPACE}" -l pokt.network/purpose=validator -o json | jq -r ".items[] | select(.status.phase != \"Pending\") | select(any(.spec.containers[]; .image | contains(\"${IMAGE_TAG}\"))) | .metadata.name")

    # Check for non-scheduled pods with incorrect image SHA to delete
    kubectl get pods -n "${NAMESPACE}" -l pokt.network/purpose=validator -o json | jq -r ".items[] | select(.status.phase == \"Pending\") | select(any(.spec.containers[]; .image | contains(\"${IMAGE_TAG}\") | not)) | .metadata.name" | while read INCORRECT_POD; do
        if [[ -n "${INCORRECT_POD}" ]]; then
            echo "Non-scheduled pod with incorrect image found: ${INCORRECT_POD}. Deleting..."
            kubectl delete pod -n "${NAMESPACE}" "${INCORRECT_POD}"
            echo "Pod deleted. StatefulSet will recreate the pod."
            # Wait for a short duration to allow the StatefulSet to recreate the pod before checking again
            sleep 10
        fi
    done

    if [[ -n "${SCHEDULED_POD}" ]]; then
        echo "Scheduled pod found: ${SCHEDULED_POD}"
        break
    else
        echo "Validator with image ${IMAGE_TAG} is not scheduled yet and no incorrect pods found. Will retry checking for scheduled or incorrect pods in 10 seconds..."
        sleep 10
    fi
done

# Create a job to run the e2e tests
echo "Creating a job to run the e2e tests..."
envsubst <.github/workflows-helpers/run-e2e-test-job-template.yaml >job.yaml
kubectl apply -f job.yaml

# Wait for the pod to be created and be in a scheduled state
echo "Waiting for the e2e test pod to be scheduled..."
while :; do
    POD_NAME=$(kubectl get pods -n "${NAMESPACE}" --selector=job-name=${JOB_NAME} -o jsonpath='{.items[*].metadata.name}')
    [[ -z "${POD_NAME}" ]] && echo "Waiting for pod to be scheduled..." && sleep 5 && continue
    POD_STATUS=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.phase}')
    if [[ "${POD_STATUS}" != "Pending" ]]; then
        echo "Current pod status: ${POD_STATUS}. Pod is scheduled."
        break
    fi
    echo "Current pod status: ${POD_STATUS}. Waiting for pod to be scheduled..."
    sleep 5
done

echo "Pod is scheduled. Monitoring logs and status..."

# Stream the pod logs in the background, including init container logs
kubectl logs -f "${POD_NAME}" -n "${NAMESPACE}" --all-containers=true &

# Monitor pod status in a loop
while :; do
    CURRENT_STATUS=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath="{.status.containerStatuses[0].state}")
    INIT_CONTAINER_STATUS=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath="{.status.initContainerStatuses[0].state}")
    
    if echo $INIT_CONTAINER_STATUS | grep -q 'terminated'; then
        INIT_EXIT_CODE=$(echo $INIT_CONTAINER_STATUS | jq '.terminated.exitCode')
        if [[ "$INIT_EXIT_CODE" != "0" ]]; then
            echo "Init container terminated with exit code ${INIT_EXIT_CODE}"
            kubectl delete job "${JOB_NAME}" -n "${NAMESPACE}"
            exit 1
        fi
    fi

    if echo $CURRENT_STATUS | grep -q 'terminated'; then
        EXIT_CODE=$(echo $CURRENT_STATUS | jq '.terminated.exitCode')
        if [[ "$EXIT_CODE" != "0" ]]; then
            echo "Container terminated with exit code ${EXIT_CODE}"
            kubectl delete job "${JOB_NAME}" -n "${NAMESPACE}"
            exit 1
        fi
        break
    fi
    sleep 5
done
