#!/usr/bin/env sh
set -eu

version=$(tr -d '[:space:]' < VERSION)
test "$version" = "1.4.1"
grep -q "\"version\": \"${version}\"" web/package.json
grep -q "version: ${version}" docs/contracts/api-contract.yaml

test -f docs/iterations/README.md
test -f docs/releases/CHANGELOG.md
test -f "docs/releases/v${version}.md"
grep -q 'DEV-20260810-003' docs/iterations/README.md
grep -q 'DEV-20260810-003' "docs/releases/v${version}.md"

for required in \
  docs/archive/api-contracts/MANIFEST.md \
  docs/archive/schemas/v0.4.0/MANIFEST.md \
  docs/archive/prompts/v0.4.0/MANIFEST.md \
  docs/archive/migrations/MANIFEST.md \
  docs/contracts/error-codes.md \
  docs/contracts/task-state-machine.md \
  docs/contracts/agent-tool-contract.md
do
  test -f "$required"
done

ids=$(find docs/iterations -type f -name 'DEV-*.md' -exec basename {} .md \; | sort)
test "$(printf '%s\n' "$ids" | wc -l | tr -d ' ')" = "$(printf '%s\n' "$ids" | sort -u | wc -l | tr -d ' ')"

for up in migrations/*.up.sql
do
  base=${up%.up.sql}
  test -f "${base}.down.sql"
done

if grep -R 'AutoMigrate' internal --include='*.go' >/dev/null 2>&1
then
  echo 'AutoMigrate is forbidden; use versioned SQL migrations.' >&2
  exit 1
fi

for prompt in configs/prompts/*_v1.0.0.txt
do
  grep -q '^prompt_name:' "$prompt"
  grep -q '^prompt_version: 1.0.0$' "$prompt"
  grep -q '^supported_schema_version: 1.0.0$' "$prompt"
done

for prompt in configs/prompts/business_agent_*_v1.1.0.txt
do
  grep -q '^prompt_name:' "$prompt"
  grep -q '^prompt_version: 1.1.0$' "$prompt"
  grep -q '^supported_schema_version: 1.0.0$' "$prompt"
done

for prompt in configs/prompts/business_agent_*_v1.2.0.txt
do
  grep -q '^prompt_name:' "$prompt"
  grep -q '^prompt_version: 1.2.0$' "$prompt"
  grep -q '^supported_schema_version: 1.0.0$' "$prompt"
done

verify_manifest_hashes() {
  manifest=$1
  shift
  for archived_file in "$@"
  do
    checksum=$(shasum -a 256 "$archived_file" | awk '{print $1}')
    grep -q "$checksum" "$manifest"
  done
}

verify_manifest_hashes docs/archive/api-contracts/MANIFEST.md docs/archive/api-contracts/*.yaml
verify_manifest_hashes docs/archive/schemas/v0.4.0/MANIFEST.md docs/archive/schemas/v0.4.0/*.json
verify_manifest_hashes docs/archive/prompts/v0.4.0/MANIFEST.md docs/archive/prompts/v0.4.0/*.txt
verify_manifest_hashes docs/archive/migrations/MANIFEST.md migrations/*.sql

echo "archive_check=ok version=${version} iterations=$(printf '%s\n' "$ids" | wc -l | tr -d ' ')"
