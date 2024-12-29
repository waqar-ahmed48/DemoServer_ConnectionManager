#!/bin/bash

VAULT_POD_NAME=$(kubectl get pods -n vault-ns -l app.kubernetes.io/name=vault -o jsonpath="{.items[0].metadata.name}")


MINIKUBE_VAULT_SECRETS=$(kubectl exec $VAULT_POD_NAME --namespace vault-ns -- vault operator init)

MINIKUBE_VAULT_UNSEAL_KEY_1=$(echo "$MINIKUBE_VAULT_SECRETS" | grep -oP 'Unseal Key 1: \K[^ ]+')
MINIKUBE_VAULT_UNSEAL_KEY_2=$(echo "$MINIKUBE_VAULT_SECRETS" | grep -oP 'Unseal Key 2: \K[^ ]+')
MINIKUBE_VAULT_UNSEAL_KEY_3=$(echo "$MINIKUBE_VAULT_SECRETS" | grep -oP 'Unseal Key 3: \K[^ ]+')
MINIKUBE_VAULT_UNSEAL_KEY_4=$(echo "$MINIKUBE_VAULT_SECRETS" | grep -oP 'Unseal Key 4: \K[^ ]+')
MINIKUBE_VAULT_UNSEAL_KEY_5=$(echo "$MINIKUBE_VAULT_SECRETS" | grep -oP 'Unseal Key 5: \K[^ ]+')
MINIKUBE_VAULT_ROOT_TOKEN=$(echo "$MINIKUBE_VAULT_SECRETS" | grep -oP 'Initial Root Token: \K[^ ]+')

kubectl exec $VAULT_POD_NAME -n vault-ns -- vault operator unseal $MINIKUBE_VAULT_UNSEAL_KEY_1
kubectl exec $VAULT_POD_NAME -n vault-ns -- vault operator unseal $MINIKUBE_VAULT_UNSEAL_KEY_2
kubectl exec $VAULT_POD_NAME -n vault-ns -- vault operator unseal $MINIKUBE_VAULT_UNSEAL_KEY_3

kubectl exec $VAULT_POD_NAME -n vault-ns -- vault login $MINIKUBE_VAULT_ROOT_TOKEN > /dev/null

kubectl exec $VAULT_POD_NAME --namespace vault-ns -- vault auth enable -default-lease-ttl=1m -max-lease-ttl=5m approle

VAULT_DEMOSERVER_POLICY=$(cat "./vault/demoserver.hcl")

kubectl exec $VAULT_POD_NAME --namespace vault-ns -- /bin/sh -c "echo '$VAULT_DEMOSERVER_POLICY' | vault policy write demoserver -"

kubectl exec $VAULT_POD_NAME --namespace vault-ns -- vault write auth/approle/role/demoserver token_policies="demoserver"

DEMOSERVER_CONNECTIONMANAGER_VAULT_ROLE_ID=$(kubectl exec $VAULT_POD_NAME --namespace vault-ns -- vault read auth/approle/role/demoserver/role-id | grep 'role_id ' | awk '{print $2}')
DEMOSERVER_CONNECTIONMANAGER_VAULT_SECRET_ID=$(kubectl exec $VAULT_POD_NAME --namespace vault-ns -- vault write -force auth/approle/role/demoserver/secret-id | grep 'secret_id ' | awk '{print $2}')

kubectl create secret -n demoserver generic demoserver-connectionmanager \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_HOST=my-postgres-release-postgresql-ha-pgpool.demoserver.svc.cluster.local \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_PORT=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_PORT} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_USERNAME=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_USERNAME} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_USERNAME=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_USERNAME} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_PASSWORD=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_PASSWORD} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_CONNECTIONPOOLSIZE=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_CONNECTIONPOOLSIZE} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_CONNECTIONPOOLSIZE=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_CONNECTIONPOOLSIZE} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_POSTGRES_SSLMODE=${DEMOSERVER_CONNECTIONMANAGER_POSTGRES_SSLMODE} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_VAULT_HOST=vault.vault-ns.svc.cluster.local \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_VAULT_PORT=8200 \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_VAULT_ROLE_ID=${DEMOSERVER_CONNECTIONMANAGER_VAULT_ROLE_ID} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_VAULT_SECRET_ID=${DEMOSERVER_CONNECTIONMANAGER_VAULT_SECRET_ID} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_VAULT_TLSSKIPVERIFY=${DEMOSERVER_CONNECTIONMANAGER_VAULT_TLSSKIPVERIFY} \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_VAULT_HTTPS=false \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_OTLP_HOST=my-opentelemetry-collector.jaeger-ns.svc.cluster.local \
	--from-literal=DEMOSERVER_CONNECTIONMANAGER_OTLP_PORT=4318