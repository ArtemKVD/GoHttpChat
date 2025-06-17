until PGPASSWORD=$DB_PASSWORD psql -h "$host" -U "postgres" -d "$DB_NAME" -c '\q'; do
  >&2 echo "Waiting for DB to become ready..."
  sleep 2
done