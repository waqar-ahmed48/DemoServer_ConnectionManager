#!/bin/bash

export VAULT_ADDR="https://127.0.0.1:8200"

DEMO01_ACCESS_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST01" | grep 'access_key' | awk '{print $2}')
DEMO01_SECRET_ACCESS_KEY=$(vault kv get -tls-skip-verify -mount="kv" "DEMOSERVER\AWS_DEMO01_TEST01" | grep 'secret_key' | awk '{print $2}')

if [ -z "$DEMO01_ACCESS_KEY" ]; then
  echo Environment Variable DEMO01_ACCESS_KEY is not set. Abort!!
  echo DEMO01_ACCESS_KEY: $DEMO01_ACCESS_KEY
  exit 1
fi

if [ -z "$DEMO01_SECRET_ACCESS_KEY" ]; then
  echo Environment Variable DEMO01_SECRET_ACCESS_KEY is not set. Abort!!
  echo DEMO01_SECRET_ACCESS_KEY: $DEMO01_SECRET_ACCESS_KEY
  exit 1
fi

VAULT_COMMAND_OUTPUT=$(vault write -tls-skip-verify auth/approle/login role_id="$DEMOSERVER_VAULT_ROLE_ID" secret_id="$DEMOSERVER_VAULT_SECRET_ID" | grep 'token ' | awk '{print $2}')

# Check if version was retrieved
if [[ -n "$VAULT_COMMAND_OUTPUT" && "$VAULT_COMMAND_OUTPUT" != "null" ]]; then
  export VAULT_TOKEN=$VAULT_COMMAND_OUTPUT
  echo VAULT_TOKEN=$VAULT_TOKEN

  #Generate a UUID
  DEMOSERVER_AWS_CONNECTION_ID=$(uuidgen)
  echo DEMOSERVER_AWS_CONNECTION_ID: $DEMOSERVER_AWS_CONNECTION_ID

  DEMOSERVER_AWS_ENGINE_PATH="demoserver/aws_$DEMOSERVER_AWS_CONNECTION_ID"
  echo DEMOSERVER_AWS_ENGINE_PATH: $DEMOSERVER_AWS_ENGINE_PATH

  VAULT_COMMAND_OUTPUT=$(vault secrets enable -tls-skip-verify -path=$DEMOSERVER_AWS_ENGINE_PATH aws)
  echo Enable AWS Secrets Engine: $VAULT_COMMAND_OUTPUT

  VAULT_COMMAND_OUTPUT=$(vault write -tls-skip-verify $DEMOSERVER_AWS_ENGINE_PATH/config/root access_key="$DEMO01_ACCESS_KEY" secret_key="$DEMO01_SECRET_ACCESS_KEY" region="us-east-1")
  echo Configure AWS Secrets Engine: $VAULT_COMMAND_OUTPUT

  VAULT_COMMAND_OUTPUT=$(vault write -tls-skip-verify $DEMOSERVER_AWS_ENGINE_PATH/config/lease lease="20s" lease_max="120s")
  echo Configure AWS Secrets Engine: $VAULT_COMMAND_OUTPUT

  VAULT_COMMAND_OUTPUT=$(vault write -tls-skip-verify $DEMOSERVER_AWS_ENGINE_PATH/roles/DemoUser credential_type=iam_user policy_arns="arn:aws:iam::aws:policy/AdministratorAccess")
  echo Create IAM Role in AWS Secrets Engine: $VAULT_COMMAND_OUTPUT

  VAULT_COMMAND_OUTPUT=$(vault read -tls-skip-verify $DEMOSERVER_AWS_ENGINE_PATH/creds/DemoUser )
  echo Create temp IAM user in AWS account: $VAULT_COMMAND_OUTPUT  

  export AWS_ACCESS_KEY_ID=$(echo "$VAULT_COMMAND_OUTPUT" | grep 'access_key' | awk '{print $2}')
  export AWS_SECRET_ACCESS_KEY=$(echo "$VAULT_COMMAND_OUTPUT" | grep 'secret_key' | awk '{print $2}')
  export AWS_DEFAULT_REGION="us-east-1"

  aws configure list

  echo "\nWaiting for 10 seconds..."
  sleep 10
  echo "Done waiting!"

  while :
  do
    echo "Keep running"
    echo "Press CTRL+C to exit\n\n"

      # Verify AWS CLI works
      aws ec2 describe-vpcs --query 'Vpcs[*].[VpcId,State,CidrBlock,Tags]' --output table
      
    sleep 1
  done

else
  echo "Failed to set VAULT_TOKEN."
fi
