#!/bin/bash

ACCESS_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST02" | grep 'access_key' | awk '{print $2}')
SECRET_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST02" | grep 'secret_key' | awk '{print $2}')

curl -X PATCH http://localhost:5678/v1/connectionmgmt/connection/aws/929f0618-0454-4505-94bb-9257856d2b4d \
    -H "Content-Type: application/json"  \
    -d "{\"connection\": {\"name\": \"Demo01Account_AWS_1_New\",\"description\": \"Demo01Account AWS Account description_1_New\",\"connectiontype\": \"\"}, \"accesskey\": \"$ACCESS_KEY\", \"secretaccesskey\": \"$SECRET_KEY\", \"default_region\": \"us-west-1\", \"default_lease_ttl\": \"30s\", \"max_lease_ttl\": \"70s\", \"role_name\": \"DemoUser\", \"credential_type\": \"iam_user\", \"policy_arns\": [\"arn:aws:iam::aws:policy/AdministratorAccess\"]}" | jq