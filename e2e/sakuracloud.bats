#!/usr/bin/env bats

KIND_CLUSTER_NAME=kind
SAKURACLOUD_ZONE="tk1a"
PROVIDER_NAMESPACE="kube-system"
PROVIDER_DOCKER_IMAGE="secrets-store-csi-driver-provider-sakuracloud:test"
NAMESPACE="default"
TEST_ID="${TEST_ID:-0}"
export SECRET1_NAME="test1-${TEST_ID}"
export SECRET2_NAME="test2-${TEST_ID}"

setup_file() {
  # Install Secrets Store CSI Driver
  helm repo add secrets-store-csi-driver https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts
  helm --namespace $PROVIDER_NAMESPACE install csi-secrets-store secrets-store-csi-driver/secrets-store-csi-driver \
    --replace \
    --set enableSecretRotation=true \
    --set rotationPollInterval=15s \
    --set syncSecret.enabled=true

  # Create sakuracloud credential
  if [[ -z "$SAKURACLOUD_ACCESS_TOKEN" || -z "$SAKURACLOUD_ACCESS_TOKEN_SECRET" ]]; then
    echo "SAKURACLOUD_ACCESS_TOKEN and SAKURACLOUD_ACCESS_TOKEN_SECRET must be set."
    exit 1
  fi
  kubectl create secret generic sakuracloud-credentials \
    --namespace $PROVIDER_NAMESPACE \
    --from-literal access-token=$SAKURACLOUD_ACCESS_TOKEN \
    --from-literal access-token-secret=$SAKURACLOUD_ACCESS_TOKEN_SECRET

  # Build and load the provider image
  docker build -t $PROVIDER_DOCKER_IMAGE ..
  kind load docker-image --name $KIND_CLUSTER_NAME $PROVIDER_DOCKER_IMAGE

  # Create test secrets
  if [[ -z "$SAKURACLOUD_VAULT_ID" ]]; then
    echo "SAKURACLOUD_VAULT_ID must be set."
    exit 1
  fi
  echo "Creating test secrets with ID: $TEST_ID"
  usacloud rest request --zone $SAKURACLOUD_ZONE /secretmanager/vaults/$SAKURACLOUD_VAULT_ID/secrets -XPOST -d'{"Secret":{"Name": "'$SECRET1_NAME'", "Value": "test1value"}}'
  usacloud rest request --zone $SAKURACLOUD_ZONE /secretmanager/vaults/$SAKURACLOUD_VAULT_ID/secrets -XPOST -d'{"Secret":{"Name": "'$SECRET2_NAME'", "Value": "test2value"}}'
}

teardown_file() {
  # Uninstall Secrets Store CSI Driver
  helm --namespace $PROVIDER_NAMESPACE uninstall csi-secrets-store

  # Remove the sakuracloud credentials secret
  kubectl delete secret sakuracloud-credentials --namespace $PROVIDER_NAMESPACE

  # Remove the provider image
  docker rmi $PROVIDER_DOCKER_IMAGE

  # Delete test pods
  kubectl delete --namespace $NAMESPACE pods --all --force

  # Delete test secrets
  echo "Deleting test secrets with ID: $TEST_ID"
  usacloud rest request --zone $SAKURACLOUD_ZONE /secretmanager/vaults/$SAKURACLOUD_VAULT_ID/secrets -XDELETE -d'{"Secret":{"Name": "'$SECRET1_NAME'"}}'
  usacloud rest request --zone $SAKURACLOUD_ZONE /secretmanager/vaults/$SAKURACLOUD_VAULT_ID/secrets -XDELETE -d'{"Secret":{"Name": "'$SECRET2_NAME'"}}'
}

@test "install sakuracloud provider" {
  # install sakuracloud provider
  PROVIDER_DOCKER_IMAGE=$PROVIDER_DOCKER_IMAGE envsubst < manifest/installer.yaml | kubectl apply --server-side -f -

  # wait for pods
  kubectl wait --for condition=Ready --timeout 60s pods --namespace $PROVIDER_NAMESPACE -l app=secrets-store-csi-driver-provider-sakuracloud
}

@test "deploy secretproviderclass" {
  kubectl wait --for condition=Established --timeout 60s crd secretproviderclasses.secrets-store.csi.x-k8s.io
  envsubst < manifest/secretproviderclass.yaml | kubectl apply --server-side -f -
}

@test "deploy csi inline volume pod" {
  kubectl replace --force -f manifest/pod-secrets-store-inline.yaml
  kubectl wait --for condition=Ready --timeout 60s pod --namespace $NAMESPACE secrets-store-inline

  run kubectl exec --namespace $NAMESPACE secrets-store-inline -- cat /mnt/secrets-store/$SECRET1_NAME
  [[ "${output//$'\r'}" == "test1value" ]]
}

@test "rotate secrets" {
  usacloud rest request --zone $SAKURACLOUD_ZONE /secretmanager/vaults/$SAKURACLOUD_VAULT_ID/secrets -XPOST -d'{"Secret":{"Name": "'${SECRET1_NAME}'", "Value": "test1value-updated"}}'
  sleep 30

  run kubectl exec --namespace $NAMESPACE secrets-store-inline -- cat /mnt/secrets-store/${SECRET1_NAME}
  [[ "${output//$'\r'}" == "test1value-updated" ]]
}
