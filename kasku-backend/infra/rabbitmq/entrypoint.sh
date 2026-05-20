#!/bin/sh
# Custom RabbitMQ entrypoint: starts broker then demotes the default user
# from 'administrator' to 'monitoring' so the service account cannot use
# the management API for administrative operations (principle of least privilege).
set -e

# Start RabbitMQ using the official image entrypoint in the background.
/usr/local/bin/docker-entrypoint.sh rabbitmq-server &
MQ_PID=$!

# Wait until the 'rabbit' application is fully started (not just the node).
# 'list_users' requires the rabbit app — safer than 'ping' which only checks the node.
echo "[entrypoint] Waiting for RabbitMQ to be ready..."
until rabbitmqctl list_users >/dev/null 2>&1; do
    sleep 3
done
echo "[entrypoint] RabbitMQ ready."

# Ensure the service user exists and has the minimum required privileges.
# 'monitoring' tag allows Prometheus scraping but NOT management-API admin operations.
MQ_USER="${RABBITMQ_DEFAULT_USER:-kasku_mq_user}"
MQ_PASS="${RABBITMQ_DEFAULT_PASS}"

if rabbitmqctl list_users | grep -qF "${MQ_USER}"; then
    echo "[entrypoint] User '${MQ_USER}' already exists."
else
    echo "[entrypoint] Creating user '${MQ_USER}'..."
    rabbitmqctl add_user "${MQ_USER}" "${MQ_PASS}"
fi

# Set tag to 'monitoring' only (not 'administrator') — principle of least privilege.
rabbitmqctl set_user_tags "${MQ_USER}" monitoring
echo "[entrypoint] User '${MQ_USER}' tags set to: monitoring"

# Ensure vhost permissions are set (idempotent — safe to re-run on restart).
rabbitmqctl set_permissions -p / "${MQ_USER}" ".*" ".*" ".*"
echo "[entrypoint] Vhost permissions confirmed for '${MQ_USER}'."

# Hand control back to the broker process.
wait "${MQ_PID}"
