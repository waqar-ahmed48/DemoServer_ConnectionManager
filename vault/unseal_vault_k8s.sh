#!/bin/bash

export VAULT_ADDR="https://127.0.0.1:8200"

MINIKUBE_VAULT_UNSEAL_KEY_1=$(vault kv get -tls-skip-verify -mount="kv" "MINIKUBE_VAULT" | grep 'UNSEAL_KEY_1' | awk '{print $2}')
MINIKUBE_VAULT_UNSEAL_KEY_2=$(vault kv get -tls-skip-verify -mount="kv" "MINIKUBE_VAULT" | grep 'UNSEAL_KEY_2' | awk '{print $2}')
MINIKUBE_VAULT_UNSEAL_KEY_3=$(vault kv get -tls-skip-verify -mount="kv" "MINIKUBE_VAULT" | grep 'UNSEAL_KEY_3' | awk '{print $2}')

kubectl exec -ti vault-0 --namespace vault-ns -- vault operator unseal $MINIKUBE_VAULT_UNSEAL_KEY_1
kubectl exec -ti vault-0 --namespace vault-ns -- vault operator unseal $MINIKUBE_VAULT_UNSEAL_KEY_2
kubectl exec -ti vault-0 --namespace vault-ns -- vault operator unseal $MINIKUBE_VAULT_UNSEAL_KEY_3