#!/bin/bash

while true; do
    ADDRESSES=$(kubectl get machine -o json | jq -r '.items[] | select(.metadata.labels."cluster.x-k8s.io/control-plane" != null) | .status | select(.addresses!=null) | .addresses[].address')
    if [[ $ADDRESSES != $OLD_ADDRESSES ]]; then
        cp hack/nginx.conf ./nginx.conf
        echo $ADDRESSES
        for ADDRESS in $ADDRESSES; do
            sleep 5
            echo $ADDRESS
            sed -i.bak '/upstream kubeendpoints/a\'$'\n'$'\t''server '$ADDRESS':6443 max_fails=3 fail_timeout=10s;'$'\n' nginx.conf
        done
        docker stop nginx-container || echo
        docker rm nginx-container || echo
        docker run --name=nginx-container --rm -p 8082:8082 -v $(pwd)/nginx.conf:/etc/nginx/nginx.conf nginx &
    fi
    OLD_ADDRESSES=$ADDRESSES 
    sleep 5
done


