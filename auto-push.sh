#!/bin/bash

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$REPO_DIR"

git config credential.helper store
git remote set-url origin https://github.com/shiwanshum/Dvc-Gateway.git

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
