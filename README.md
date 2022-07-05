# concurrent update test

At single row level, postgresql serializes operations, so we can be safe concurrently updating the same row concurrently if we create the proper update query.


1. Start PG

```shell
% docker-compose -f stack.yml up
```

2. Log into DB and create contexts table

```db
% psql -h localhost -d postgres -U postgres -p 5432
postgres=# create table contexts
(
id text not null constraint pk_context primary key,
version integer not null
);

CREATE TABLE
postgres=# select * from contexts;
 id | version 
----+---------
(0 filas)
```

3 . Run script and check the highest version of every context is the expected

```shell
% go run main.go
Counters: map[10:5000]
go run main.go  2,91s user 2,52s system 21% cpu 24,902 total
```
