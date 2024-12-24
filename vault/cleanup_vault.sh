#!/bin/bash

# Path to exclude
EXCLUDE_PATH="demoserver/aws_8ab80cc3-dd41-4914-bebd-adb151a08725"

# Get all mounted secrets engines
MOUNTS=$(vault secrets list -tls-skip-verify -format=json | jq -r 'keys[]')

# Iterate over each mount
for MOUNT in $MOUNTS; do
  # Check if the mount is an AWS secrets engine
  if [[ $MOUNT == demoserver/* ]]; then
    #echo "Checking mount: $MOUNT"
    if [[ $MOUNT != $EXCLUDE_PATH/ ]]; then
      echo "Deleting AWS secrets engine at path: $MOUNT"
      vault secrets disable -tls-skip-verify "$MOUNT"
    else
      echo "Skipping AWS secrets engine at path: $MOUNT"
    fi
  else
    echo "Skipping AWS secrets engine at path: $MOUNT"
  fi
done

echo "Cleanup completed."
