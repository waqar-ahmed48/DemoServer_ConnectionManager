#!/bin/bash
curl http://$(minikube ip):30283/status | jq

for i in {1..20}
do
   echo "Welcome $i times"
    curl -X POST -d @./record_example_1_with_attachment.json -H "Content-Type: application/json" http://$(minikube ip):30283/record/123456 | jq
    curl -X POST -d @./record_example_1.json -H "Content-Type: application/json" http://$(minikube ip):30283/record/123456 | jq
    curl -X POST -d @./record_example_2_with_attachment.json -H "Content-Type: application/json" http://$(minikube ip):30283/record/123456 | jq
    curl -X POST -d @./record_example_2.json -H "Content-Type: application/json" http://$(minikube ip):30283/record/123456 | jq
done

curl http://$(minikube ip):30283/recordcount/123456 | jq

