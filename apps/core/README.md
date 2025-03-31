## Setup

#### Docker

`brew install docker`

You can use docker compose directly like `docker compose up -d`

#### Postgres DB

Install migrate - `brew install golang-migrate`

Create a migration example: `migrate create -ext sql -dir migrations/postgres -seq create_users_table`

Apply migration up / down `migrate -database "postgres://user:password@localhost:5432/mydb?sslmode=disable" -path migrations/postgres up`

`migrate -database "postgres://user:password@localhost:5432/mydb?sslmode=disable" -path migrations/postgres down`

#### HTTPS Server

For local development, generate a self signed certificate using OpenSSL

```
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
```

For production, obtain an SSL certificate from Let's Encrypt or another CA.

- Install ansible `pip install ansible`
- Run `./decrypt.sh`
- Run server `air`

- NOTE: vault_pass.txt is required in your root directory
  - Get this from administrator to get the app running

#### Postgres

`docker pull postgres`

`docker run --name synergy_postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=synergy -p 5433:5432 -d postgres`

Verify using `docker ps`

Login to db `docker exec -it synergy_postgres psql -U postgres -d synergy`

Check tables `\dt`

To open pgAdmin `docker run --rm -p 5050:80 -e PGADMIN_DEFAULT_EMAIL=admin@example.com -e PGADMIN_DEFAULT_PASSWORD=postgres dpage/pgadmin4`

#### Scylla

Log into a node `docker exec -it scylla1 cqlsh`

Check cluster status `docker exec -it scylla1 nodetool status`

Check docker logs `docker logs -f scylla1`

Create a keyspace

```
CREATE KEYSPACE IF NOT EXISTS synergy WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};
```

```
DESCRIBE KEYSPACES
```

```
DESCRIBE KEYSPACE synergy
```
