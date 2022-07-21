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

CICD workflows include the build, the unit tests and the system integration tests. 
All the related docker files and docker compose files are located in the "cicd/docker" directory. 
The test scripts are in the "testscripts" directory. 

### Unit Test 
The unit test is go test ./... which assumes the presence of Postgresql, Mongo and Redis databases. All these database servers are set up to run in separate containers using the configuration in Dockerfile_unittest_env_linux and docker-compose-unittest_env.yml, respectively. 

The container ${DOCKER_PREFIX}_dfstore_unittest contains all database clients to initalize the database data required by the go test program. It will ensure that all database servers are already up before proceeding the operation. The operation is written in testscripts/unittest_init.sh. 

Once, the test setup is done, CICD process will start to build the code and run go test. 

### System Integration Test
After the unit test is completed, all containers will be stopped. Then we start the containers required to set up the environment for system integration using the configuration in Dockerfile_systest_linux and docker-compose-systest.yml, respectively. 

The container ${DOCKER_PREFIX}_dfstore_systest takes care of the system integration setup written in testscripts/systest_init.sh. 

Once, the test setup is done, CICD process will start to execute testscripts/systest.sh. 