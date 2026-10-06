#!/usr/bin/env bats

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"
PROVIDER_NAMESPACE="kube-system"
PROVIDER_DOCKER_IMAGE="secrets-store-csi-driver-provider-sakuracloud:test"
NAMESPACE="default"
TEST_ID="${TEST_ID:-0}"
export SECRET1_NAME="test1-${TEST_ID}"
export SECRET2_NAME="test2-${TEST_ID}"
SAKUMOCK_API_URL="http://127.0.0.1:18082"

sakumock_request() {
  local path="$1"
  local data="$2"
  curl --noproxy '*' --fail --silent --show-error --max-time 10 \
    --user dummy:dummy --header 'Content-Type: application/json' \
    --request POST --data "$data" "$SAKUMOCK_API_URL/secretmanager/$path"
}

setup() {
  SAKURACLOUD_VAULT_ID="$(cat "$BATS_FILE_TMPDIR/vault-id")"
  export SAKURACLOUD_VAULT_ID
}

setup_file() {
  # Start the mock API in the same cluster as the provider.
  kubectl apply --server-side -f manifest/sakumock.yaml
  kubectl rollout status --namespace "$PROVIDER_NAMESPACE" deployment/sakumock --timeout=120s
  curl --noproxy '*' --fail --silent --show-error \
    --retry 30 --retry-delay 1 --retry-all-errors --retry-max-time 30 --max-time 5 \
    "$SAKUMOCK_API_URL/secretmanager/vaults" > /dev/null

  # Create a fresh vault and secrets without using account credentials.
  local vault
  vault="$(sakumock_request vaults \
    "{\"Vault\":{\"Name\":\"e2e-${TEST_ID}\",\"KmsKeyID\":\"dummy-kms-key\"}}")"
  local vault_id
  vault_id="$(jq --exit-status --raw-output '.Vault.ID | select(type == "string" and length > 0)' <<< "$vault")"
  echo "$vault_id" > "$BATS_FILE_TMPDIR/vault-id"
  sakumock_request "vaults/$vault_id/secrets" \
    "{\"Secret\":{\"Name\":\"$SECRET1_NAME\",\"Value\":\"test1value\"}}"
  sakumock_request "vaults/$vault_id/secrets" \
    "{\"Secret\":{\"Name\":\"$SECRET2_NAME\",\"Value\":\"test2value\"}}"

  # Install Secrets Store CSI Driver
  helm repo add secrets-store-csi-driver https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts
  helm --namespace "$PROVIDER_NAMESPACE" install csi-secrets-store secrets-store-csi-driver/secrets-store-csi-driver \
    --replace \
    --set enableSecretRotation=true \
    --set rotationPollInterval=15s \
    --set syncSecret.enabled=true

  # Always use dummy credentials, even if real credentials are set locally.
  kubectl create secret generic sakuracloud-credentials \
    --namespace "$PROVIDER_NAMESPACE" \
    --from-literal access-token=dummy \
    --from-literal access-token-secret=dummy

  # Build and load the provider image
  docker build -t "$PROVIDER_DOCKER_IMAGE" ..
  kind load docker-image --name "$KIND_CLUSTER_NAME" "$PROVIDER_DOCKER_IMAGE"
}

teardown_file() {
  local cleanup_status=0
  kubectl delete --namespace "$NAMESPACE" pod secrets-store-inline --ignore-not-found || cleanup_status=1
  kubectl delete --namespace "$NAMESPACE" secretproviderclass basic-test --ignore-not-found || cleanup_status=1
  kubectl delete -k manifest/installer --ignore-not-found || cleanup_status=1

  # Removing the mock also discards all test vaults and secrets.
  kubectl delete -f manifest/sakumock.yaml --ignore-not-found || cleanup_status=1

  # Uninstall Secrets Store CSI Driver
  if helm --namespace "$PROVIDER_NAMESPACE" status csi-secrets-store >/dev/null 2>&1; then
    helm --namespace "$PROVIDER_NAMESPACE" uninstall csi-secrets-store || cleanup_status=1
  fi

  # Remove the sakuracloud credentials secret
  kubectl delete secret sakuracloud-credentials --namespace "$PROVIDER_NAMESPACE" --ignore-not-found || cleanup_status=1

  # Remove the provider image
  if docker image inspect "$PROVIDER_DOCKER_IMAGE" >/dev/null 2>&1; then
    docker rmi "$PROVIDER_DOCKER_IMAGE" || cleanup_status=1
  fi
  return "$cleanup_status"
}

@test "install sakuracloud provider" {
  # install sakuracloud provider
  kubectl apply --server-side -k manifest/installer

  # wait for pods
  kubectl wait --for condition=Ready --timeout 60s pods --namespace "$PROVIDER_NAMESPACE" -l app=secrets-store-csi-driver-provider-sakuracloud
}

@test "deploy secretproviderclass" {
  kubectl wait --for condition=Established --timeout 60s crd secretproviderclasses.secrets-store.csi.x-k8s.io
  envsubst < manifest/secretproviderclass.yaml | kubectl apply --server-side -f -
}

@test "deploy csi inline volume pod" {
  kubectl replace --force -f manifest/pod-secrets-store-inline.yaml
  kubectl wait --for condition=Ready --timeout 60s pod --namespace "$NAMESPACE" secrets-store-inline

  run kubectl exec --namespace "$NAMESPACE" secrets-store-inline -- cat "/mnt/secrets-store/$SECRET1_NAME"
  [[ "$status" -eq 0 ]]
  [[ "${output//$'\r'}" == "test1value" ]]
}

@test "rotate secrets" {
  sakumock_request "vaults/$SAKURACLOUD_VAULT_ID/secrets" \
    "{\"Secret\":{\"Name\":\"$SECRET1_NAME\",\"Value\":\"test1value-updated\"}}"
  # CSI Driver 1.6+ rotates on kubelet republish calls, which can take over a minute.
  for ((attempt = 0; attempt < 90; attempt++)); do
    run kubectl exec --namespace "$NAMESPACE" secrets-store-inline -- cat "/mnt/secrets-store/$SECRET1_NAME"
    if [[ "$status" -eq 0 && "${output//$'\r'}" == "test1value-updated" ]]; then
      return 0
    fi
    sleep 2
  done

  echo "Secret did not rotate within 180 seconds. Last result: $output"
  kubectl logs --namespace "$PROVIDER_NAMESPACE" -l app=secrets-store-csi-driver-provider-sakuracloud --tail=80
  kubectl logs --namespace "$PROVIDER_NAMESPACE" -l app=secrets-store-csi-driver --all-containers=true --tail=80
  [[ "$status" -eq 0 ]]
  [[ "${output//$'\r'}" == "test1value-updated" ]]
}
