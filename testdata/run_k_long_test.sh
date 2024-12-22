#!/bin/bash
curl http://$(minikube ip):30283/status

for i in {1..200}
do
    echo "Welcome $i times"
    now=$(date +"%T")
    echo "Current time : $now"

    curl -X POST -d @./record_example_1_with_attachment.json -H "Content-Type: application/json" http://$(minikube ip):30283/record/123456
    
    sleep 600s
done


