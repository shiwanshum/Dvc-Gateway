#!/bin/bash

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$REPO_DIR"

CRED_FILE="../cred.txt"

# Extract GitHub credentials from cred.txt
GIT_USER=$(grep -A 2 "#============ github" "$CRED_FILE" | grep "username:" | awk -F': ' '{print $2}' | tr -d '\r')
GIT_PASS=$(grep -A 2 "#============ github" "$CRED_FILE" | grep "password:" | awk -F': ' '{print $2}' | tr -d '\r')

if [ -z "$GIT_USER" ] || [ -z "$GIT_PASS" ]; then
    echo "Error: Could not extract GitHub credentials from $CRED_FILE"
    exit 1
fi

# Set the remote URL dynamically without hardcoding it in the script
REMOTE_URL="https://${GIT_USER}:${GIT_PASS}@github.com/${GIT_USER}/Dvc-Gateway.git"
git remote set-url origin "$REMOTE_URL"

echo "Watching for changes in $REPO_DIR ..."

while true; do
  inotifywait -r -e modify,create,delete,move --exclude '\.git' "$REPO_DIR" 2>/dev/null

  sleep 2

  git add -A
  CHANGES=$(git diff --cached --name-only)
  if [ -n "$CHANGES" ]; then
    echo "Changes detected: $CHANGES"
    git commit -m "Auto commit: $(date '+%Y-%m-%d %H:%M:%S')" --quiet
    git push origin main 2>&1
    echo "Pushed to GitHub at $(date '+%Y-%m-%d %H:%M:%S')"
  fi
done
