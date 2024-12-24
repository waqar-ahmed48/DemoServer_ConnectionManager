#!/bin/bash

export VAULT_ADDR="https://127.0.0.1:8200"

ACCESS_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST01" | grep 'access_key' | awk '{print $2}')
SECRET_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST01" | grep 'secret_key' | awk '{print $2}')

curl -X POST http://localhost:5678/v1/connectionmgmt/connection/aws \
    -H "Content-Type: application/json"  \
    -d "{\"connection\": {\"name\": \"Demo01Account_AWS_3\",\"description\": \"Demo01Account AWS Account description_3\",\"connectiontype\": \"\"}, \"accesskey\": \"$ACCESS_KEY\", \"secretaccesskey\": \"$SECRET_KEY\", \"default_region\": \"us-east-1\", \"default_lease_ttl\": \"20s\", \"max_lease_ttl\": \"60s\", \"role_name\": \"DemoUser\", \"credential_type\": \"iam_user\", \"policy_arns\": [\"arn:aws:iam::aws:policy/AdministratorAccess\"]}" | jq