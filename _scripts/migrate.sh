ENV="dev"
ENDPOINT="grpcs://ydb.serverless.yandexcloud.net:2135"

DB_PATH="
{
  \"dev\": {
    \"source\": \"/REGION/XXX/YYY\",
    \"target\": \"/REGION/XXX/YYY\"
  },
  \"prod\": {
    \"source\": \"/REGION/XXX/YYY\",
    \"target\": \"/REGION/XXX/YYY\"
  }
}
"

echo "[INFO] Using environment: $ENV"

DB_PATH_SOURCE=$(echo -n "$DB_PATH" | jq -r ".\"$ENV\".source")
DB_PATH_TARGET=$(echo -n "$DB_PATH" | jq -r ".\"$ENV\".target")

if [ -z "$DB_PATH_SOURCE" ] || [ -z "$DB_PATH_TARGET" ]; then
  echo "[ERROR] Invalid environment: $ENV"
  exit 1
fi

echo "[INFO] Source database path: \"$DB_PATH_SOURCE\""
echo "[INFO] Target database path: \"$DB_PATH_TARGET\""

ydb config profile replace source-db -d "$DB_PATH_SOURCE" -e "$ENDPOINT" --token-file "./source.txt"
echo "[INFO] Source database profile created"

ydb config profile replace target-db -d "$DB_PATH_TARGET" -e "$ENDPOINT" --token-file "./target.txt"
echo "[INFO] Target database profile created"

DIR="$(eval echo ~$USER)/ydb_dump/$ENV/chats"
echo "[INFO] Using directory: $DIR"

echo "[INFO] Recreating directory: $DIR"
rm -rf "$DIR"
mkdir -p "$DIR"

echo "[INFO] Dumping source database..."
ydb --profile source-db tools dump --path chats --output "$DIR"

echo "[INFO] Restoring to target database..."
ydb --profile target-db tools restore --path "$DB_PATH_TARGET" --input "$DIR"
