#!/bin/bash
# Unified PostgreSQL/TimescaleDB Backup and Restore Script

COMMAND=$1
FILE=$2
POSTGRES_POD=$(kubectl get pod -l app=postgres -o jsonpath="{.items[0].metadata.name}")

if [ -z "$POSTGRES_POD" ]; then
  echo "Error: Postgres pod not found."
  exit 1
fi

if [ "$COMMAND" == "backup" ]; then
  BACKUP_FILE="backup_$(date +%Y%m%d_%H%M%S).sql"
  echo "Starting backup of gatewaydb on pod $POSTGRES_POD..."
  kubectl exec "$POSTGRES_POD" -- pg_dump -U admin -d gatewaydb -F c > "$BACKUP_FILE"
  echo "Backup successfully saved to $BACKUP_FILE"

elif [ "$COMMAND" == "restore" ]; then
  if [ -z "$FILE" ]; then
    echo "Error: Please specify the backup file to restore."
    echo "Usage: ./backup_restore.sh restore <filename>"
    exit 1
  fi
  
  if [ ! -f "$FILE" ]; then
    echo "Error: File $FILE does not exist."
    exit 1
  fi
  
  echo "Starting restore from $FILE to gatewaydb on pod $POSTGRES_POD..."
  
  # Copy file to pod first
  kubectl cp "$FILE" "$POSTGRES_POD:/tmp/restore.sql"
  
  # Perform restore
  kubectl exec "$POSTGRES_POD" -- pg_restore -U admin -d gatewaydb --clean --if-exists /tmp/restore.sql
  
  echo "Restore completed successfully."

else
  echo "Usage: ./backup_restore.sh [backup | restore <file>]"
  exit 1
fi
