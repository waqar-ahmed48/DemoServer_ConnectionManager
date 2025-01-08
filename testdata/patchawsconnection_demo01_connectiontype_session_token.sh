#!/bin/bash

ACCESS_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST02" | grep 'access_key' | awk '{print $2}')
SECRET_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST02" | grep 'secret_key' | awk '{print $2}')
CONNECTIONID=$(curl -s http://localhost:5678/v1/connectionmgmt/connections/aws | jq -r '.awsconnections[0].id') 

curl -X PATCH http://localhost:5678/v1/connectionmgmt/connection/aws/$CONNECTIONID \
    -H "Content-Type: application/json"  \
    -d "{\"connection\": {\"name\": \"Demo01Account_AWS_1_New\",\"description\": \"Demo01Account AWS Account description_1_New\",\"connectiontype\": \"\"}, \"accesskey\": \"$ACCESS_KEY\", \"secretaccesskey\": \"$SECRET_KEY\", \"default_region\": \"us-west-2\", \"default_lease_ttl\": \"\", \"max_lease_ttl\": \"\", \"role_name\": \"DemoUser\", \"credential_type\": \"session_token\", \"policy_arns\": []}" | jq