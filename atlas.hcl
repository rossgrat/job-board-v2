env "local" {
  src = "file://db/schema"
  dev = "docker://postgres/18/dev?search_path=public"
  migration {
    dir = "file://db/migrations"
  }
  url = "postgres://${getenv("POSTGRES_USER")}:${getenv("POSTGRES_PASSWORD")}@${getenv("POSTGRES_HOST")}:5432/${getenv("POSTGRES_DB")}?sslmode=disable"
}
