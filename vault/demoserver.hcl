# Allow the user to enable a new AWS secrets engine
path "sys/mounts/demoserver/*" {
  capabilities = ["create", "update"]
}
# Grant full access to secret engines starting with "demoserver/"
path "demoserver/*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
