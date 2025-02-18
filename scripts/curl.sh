#!/bin/bash

create_node() {
    read -p "Enter node name: " NODE_NAME
    read -p "Enter labels (e.g. name=value) (comma separated): " LABELS

    if [ -z "$LABELS" ]; then
      LABELS_JSON="[]"
    else
      LABELS_JSON=$(echo "$LABELS" | sed 's/,/","/g' | sed 's/^/["/' | sed 's/$/"]/')
    fi

    PAYLOAD=$(cat <<EOF
{
    "apiVersion": "v1",
    "kind": "Node",
    "metadata": {
        "name": "$NODE_NAME",
        "labels": $LABELS_JSON
    }
}
EOF
)

    curl -i -X POST http://localhost:6443/nodes -d "$PAYLOAD"
}

list_nodes() {
    curl -i http://localhost:6443/nodes
}

create_pod() {
    read -p "Enter pod name: " POD_NAME
    read -p "Enter container image (e.g. nginx:latest): " CONTAINER_IMAGE
    read -p "Enter container name (e.g. nginx): " CONTAINER_NAME
    read -p "Enter labels (e.g. name=value) (comma separated): " LABELS

    if [ -z "$LABELS" ]; then
      LABELS_JSON="[]"
    else
      LABELS_JSON=$(echo "$LABELS" | sed 's/,/","/g' | sed 's/^/["/' | sed 's/$/"]/')
    fi

    PAYLOAD=$(cat <<EOF
{
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "name": "$POD_NAME",
        "labels": $LABELS_JSON
    },
    "spec": {
        "containers": [
            {
                "image": "$CONTAINER_IMAGE",
                "name": "$CONTAINER_NAME"
            }
        ]
    }
}
EOF
)

    curl -X POST http://localhost:6443/pods -d "$PAYLOAD"
}

list_pods() {
    curl -i http://localhost:6443/pods
}

update_node_status() {
    read -p "Enter node name to update status: " NODE_NAME
    read -p "Enter node IntenalIP: " NODE_IP

    PAYLOAD=$(cat <<EOF
{
    "addresses": [
        {
            "address": "$NODE_IP",
            "type": "InternalIP"
        }
    ]
}
EOF
)
    curl -X PATCH http://localhost:6443/nodes/$NODE_NAME/status -d "$PAYLOAD"
}

create_deployment() {
    read -p "Enter deployment name: " DEPLOYMENT_NAME
    read -p "Enter replicas: " REPLICAS
    read -p "Enter container image (e.g. nginx:latest): " CONTAINER_IMAGE
    read -p "Enter container name (e.g. nginx): " CONTAINER_NAME
    read -p "Enter selector labels (e.g. name=value) (comma separated): " LABELS

    if [ -z "$LABELS" ]; then
      LABELS_JSON="[]"
    else
      LABELS_JSON=$(echo "$LABELS" | sed 's/,/","/g' | sed 's/^/["/' | sed 's/$/"]/')
    fi

    PAYLOAD=$(cat <<EOF
{
    "apiVersion": "v1",
    "kind": "Deployment",
    "metadata": {
        "name": "$DEPLOYMENT_NAME"
    },
    "spec": {
        "replicas": $REPLICAS,
        "selector": {
            "matchLabels": $LABELS_JSON
        },
        "template": {
            "metadata": {
                "labels": $LABELS_JSON
            },
            "spec": {
                "containers": [
                    {
                        "image": "$CONTAINER_IMAGE",
                        "name": "$CONTAINER_NAME"
                    }
                ]
            }
        }
    }
}
EOF
)

    curl -X POST http://localhost:6443/deployments -d "$PAYLOAD"
}

echo "Choose an action:"
echo "1) Create a node"
echo "2) Update node status"
echo "3) List nodes"
echo "4) Create a pod"
echo "5) List pods"
echo "6) Create deployment"

read -p "Enter your choice: " CHOICE

case $CHOICE in
    1) create_node ;;
    2) update_node_status ;;
    3) list_nodes ;;
    4) create_pod ;;
    5) list_pods ;;
    6) create_deployment ;;
    *) echo "Invalid choice, please try again." ;;
esac

