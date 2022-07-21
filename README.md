## dfstore 

This is a modified package from the original dfstore. It adds the API for the query string input. 
The query (read) API supports (1) the nested query condition (2) select columns to display 

Examples of nested query condition:
`(([[title]] != {"Blue Train"}) AND ([[artist]] != {"John Coltrane"})) OR ([[hardcover]] == {"true"})) OR ([[year]] IN {"2018", "2022"})`,

where 
() It uses parentheses to group the logic
[[table_filedname]]
{} ... value (operand) for the operator
operator = "==", "!=", ">=", "<=", "<", ">", IN
aggregator = AND, OR

How to test this API in shown at the end of this readme

--- Original dfstrore explanation 

This is an experimental package to implement a database agnostic API that supports various backend databases such as redis, mongodb, PostgreSQL and other SQL databases using the abstraction based on dataframe grids often used in data science projects that depend on python pandas or scala RDD or similar.  Use of dataframe in Go language implementation of dfstore depends on gota at 	

https://github.com/go-gota/gota

https://pkg.go.dev/github.com/go-gota/gota/dataframe#pkg-examples


## Example usage

### dfstore_test.go

1. Contains test code for postgresql/default usage
2. test code for redis/memory use case
3. test code for document/mongodb use case

### dataframe 

```
dataRows = [][]string{
		{"title", "artist", "price"},
		{"Blue Train", "John Coltrane", "56.99"},
		{"Giant Steps", "John Coltrane", "63.99"},
		{"Jeru", "Gerry Mulligan", "17.99"},
		{"Sarah Vaughan", "Sarah Vaughan", "34.98"},
	}
```

### Open databases

```
	dfs, err := dfstore.New(context.TODO(), dbtype)
	if err != nil {
		t.Errorf("cannot get new dfstore, %v",err)
		return
	}
	defer dfs.Close()
```

### write databases

```
	err = dfs.WriteRecords(dataRows)
	if err != nil {
		t.Errorf("cannot write, %v", err)
	}
```

### read databases

```
	filters := []dataframe.F{
		dataframe.F{Colname: "artist", Comparator: series.Eq, Comparando: "John Coltrane"},
		dataframe.F{Colname: "price", Comparator: series.Greater, Comparando: "60"},
	}
	res, err := dfs.ReadRecords(filters, 20)
```

### read database using query string 

```

columns := []string{"artist", "year", "title", "hardcover"}
as := []string{"ARTIST", "YEAR", "TITLE", "HARDCOVER?"}
condition := `(([[title]] != {"Blue Train"}) AND ([[artist]] != {"John Coltrane"})) OR ([[hardcover]] == {"true"})) OR ([[year]] == {"2018"})`,
 
res, err := dfs.ReadRecordsString(columns, as, condition, 20)

```
		

## Testing

### Running a test PostgreSQL server

run postgresql server in a docker

```
docker run --rm --name postgresql -e POSTGRES_PASSWORD=password -p 5432:5432  -e POSTGRES_USER=pguser -e POSTGRES_DB=testdb postgres
```

run postgres cli in a docker
```
docker exec -it postgresql psql -h localhost -p 5432 -U pguser -W -d postgres
```

run the following in psql cli to create schema table
```
# CREATE DATABASE dfstore1;
# \c dfstore1;
# CREATE TABLE IF NOT EXISTS schema ( tablename VARCHAR(128) PRIMARY KEY, columns VARCHAR(255) NOT NULL );
# \l
```

### running a test redis server

```
docker run --name redis --rm -p 6379:6379 -d redis
```

Run redis-cli to require password login
```
$ redis-cli -a password
# config set requirepass password
```


### Running a test mongodb server

```
docker run -d --rm --name mongo -it -p 27017:27017 -e MONGO_INITDB_ROOT_USERNAME=root -e MONGO_INITDB_ROOT_PASSWORD=rootpass  mongo
```

#### Install mongosh

https://www.mongodb.com/docs/mongodb-shell/install/

#### Run mongosh

```
$ mongosh mongodb://root:rootpass@localhost:27017

test> db.dropDatabase('test')


test> use dfstore1
switched to db dfstore1

dfstore1> db.createCollection('table1')  <--- This is to create table1 for go test -run Doc
dfstore1> db.createCollection('table2')  <--- This is to create table1 for go test -run Parse, ParseCreateDB and ParseString


dfstore1> db.getCollectionNames()

dfstore1>  db.createUser( { user: 'root',pwd: 'rootpass', roles: [ { role: "readWrite", db: "dfstore1"} ] } )

dfstore1> db.getUsers()
```

Later once the test is done, we can check the database in MogoDB using the following command. 
```
dfstore1> db.getCollection('table1').find().forEach(printjson)
{
  _id: ObjectId("000000000000000000000000"),
  artist: 'John Coltrane',
  price: 58.99,
  title: 'Blue Train'
}
{ _id: 1, title: 'Blue Train', artist: 'John Coltrane', price: 56.99 }
{ _id: 2, title: 'Giant Steps', artist: 'John Coltrane', price: 63.99 }
{ _id: 3, title: 'Jeru', artist: 'Gerry Mulligan', price: 17.99 }
{
  _id: 4,
  title: 'Sarah Vaughan',
  artist: 'Sarah Vaughan',
  price: 34.98
}

dfstore1>
```


### go test
```
go test
```
### go test a specific one
```
go test -run Memory
go test -run Doc 
go test -run Default
```
### go test for query string API
```
go test -run Parse  // to create the DB for testing & test the query API
go test -run ParseCreateDB  // to create the DB for testing 
go test -run ParseString1   // to test the query API (Must run ParseCreateDB first)
```
## CICD Testing

Try to integrate the build and unit test in the CICD. 
In the go test, it assumes the presence of Postgresql, Mongo and Redis database. Each of them runs in a separate container. We set up these containers Dockerfile_unittest_env_linux and docker-compose-unittest_env.yml to bring those database containers up in different containers inside the "deploynet" bridge network. Then run go test. 

### Found issues

The go test cannot connect to that servers from the build container. However, inside that build container, after I install proper net-tools, dnsutils, iputils-ping (it was not setup in the docker file), I can ping and nslookup those services. If I install and run mongosh, I also can connect to the mongo server. 

To test this docker environment setup locally, 
create the link of docker-compose-unittest_env.yml, Dockerfile_build_linux, Dockerfile_unittest_env_link from cicd/docker directory to repository's root directory. Then run 
```
DOCKER_PREFIX="testdfstore" docker compose -f docker-compose-unittest_env.yml up --build --detach
```