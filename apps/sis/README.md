# SIS (Student Informations System)

### Migration

```sql
migrate -database "sqlite3://./db/sis.db" -path db/migrations up
```

### Seed

```sql
sqlite3 db/sis.db < db/seeds/prod.sql
```
