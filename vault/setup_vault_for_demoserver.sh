#!/bin/bash

export VAULT_ADDR="https://127.0.0.1:8200"

VAULT_VERSION=$(vault version)

# Check if version was retrieved
if [[ -n "$VAULT_VERSION" && "$VAULT_VERSION" != "null" ]]; then
  echo "HashiCorp Vault is running version: $VAULT_VERSION"
else
  echo "Failed to fetch Vault version. Please check your Vault address and token."
fi

VAULT_COMMAND_OUTPUT=$(vault auth list -tls-skip-verify | grep 'approle')
# Check if approle is enabled. If not enable it.
if [[ -n "$VAULT_COMMAND_OUTPUT" && "$VAULT_COMMAND_OUTPUT" != "null" ]]; then
  echo "AppRole is enabled: $VAULT_COMMAND_OUTPUT"
else
  VAULT_APPROLE_ENABLED = $(vault auth enable -tls-skip-verify -default-lease-ttl=1m -max-lease-ttl=5m approle)
  echo $VAULT_COMMAND_OUTPUT
fi

VAULT_COMMAND_OUTPUT=$(vault policy write -tls-skip-verify demoserver ./demoserver.hcl)
echo Setup ACL Policy: $VAULT_COMMAND_OUTPUT

VAULT_COMMAND_OUTPUT=$(vault write -tls-skip-verify auth/approle/role/demoserver token_policies="demoserver")
echo Create App Role: $VAULT_COMMAND_OUTPUT

export DEMOSERVER_CONNECTIONMANAGER_VAULT_ROLE_ID=$(vault read -tls-skip-verify auth/approle/role/demoserver/role-id | grep 'role_id ' | awk '{print $2}') 
echo role-id=$DEMOSERVER_CONNECTIONMANAGER_VAULT_ROLE_ID

export DEMOSERVER_CONNECTIONMANAGER_VAULT_SECRET_ID=$(vault write -tls-skip-verify -force auth/approle/role/demoserver/secret-id | grep 'secret_id ' | awk '{print $2}')
echo secret-id=$DEMOSERVER_CONNECTIONMANAGER_VAULT_SECRET_ID